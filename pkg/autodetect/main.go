package autodetect

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	osimagev1 "github.com/openshift/api/image/v1"
	imagereference "github.com/openshift/library-go/pkg/image/reference"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	authenticationapi "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/tracing"
)

var listenedGroupsMap = map[string]bool{"logging.openshift.io": true, "kafka.strimzi.io": true, "route.openshift.io": true}

// Background represents a procedure that runs in the background, periodically auto-detecting features
type Background struct {
	cl       client.Client
	clReader client.Reader
	dcl      discovery.DiscoveryInterface
	ticker   *time.Ticker

	firstRun         *sync.Once
	retryDetectKafka bool
	retryDetectEs    bool
}

// New creates a new auto-detect runner
func New(mgr manager.Manager) (*Background, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	return WithClients(mgr.GetClient(), dcl, mgr.GetAPIReader()), nil
}

// WithClients builds a new Background with the provided clients
func WithClients(cl client.Client, dcl discovery.DiscoveryInterface, clr client.Reader) *Background {
	// whether we should keep adjusting depending on the environment
	retryDetectEs := viper.GetString("es-provision") == v1.FlagProvisionElasticsearchAuto
	retryDetectKafka := viper.GetString("kafka-provision") == v1.FlagProvisionKafkaAuto

	return &Background{
		cl:               cl,
		dcl:              dcl,
		clReader:         clr,
		retryDetectKafka: retryDetectKafka,
		retryDetectEs:    retryDetectEs,
		firstRun:         &sync.Once{},
	}
}

// Start initializes the auto-detection process that runs in the background
func (b *Background) Start() {
	// periodically attempts to auto detect all the capabilities for this operator
	b.ticker = time.NewTicker(5 * time.Second)
	b.autoDetectCapabilities()
	log.Log.V(-1).Info("finished the first auto-detection")

	go func() {
		for {
			<-b.ticker.C
			b.autoDetectCapabilities()
		}
	}()
}

// Stop causes the background process to stop auto detecting capabilities
func (b *Background) Stop() {
	b.ticker.Stop()
}

func (b *Background) autoDetectCapabilities() {
	ctx := context.Background()

	apiList, err := AvailableAPIs(b.dcl, listenedGroupsMap)
	if err != nil {
		log.Log.Error(
			err,
			"failed to determine the platform capabilities, auto-detected properties will remain the same until next cycle.",
		)
	} else {
		b.firstRun.Do(func() {
			// the platform won't change during the execution of the operator, need to run it only once
			b.detectPlatform(ctx, apiList)

			// the version of the APIs provided by the platform will not change
			b.detectCronjobsVersion(ctx)
			b.detectAutoscalingVersion(ctx)
		})
		b.detectOAuthProxyImageStream(ctx)
		b.detectElasticsearch(ctx, apiList)
		b.detectKafka(ctx, apiList)
	}
	b.detectClusterRoles(ctx)
}

func (b *Background) detectCronjobsVersion(ctx context.Context) {
	apiGroupVersions := []string{v1.FlagCronJobsVersionBatchV1, v1.FlagCronJobsVersionBatchV1Beta1}
	detectedVersion := ""

	for _, apiGroupVersion := range apiGroupVersions {
		groupAPIList, err := b.dcl.ServerResourcesForGroupVersion(apiGroupVersion)
		if err != nil {
			log.Log.V(-1).Info(
				fmt.Sprintf("error getting %s api list: %s", apiGroupVersion, err),
			)
			continue
		}
		for _, api := range groupAPIList.APIResources {
			if api.Name == "cronjobs" {
				detectedVersion = apiGroupVersion
				break
			}
		}
	}

	if detectedVersion == "" {
		log.Log.V(2).Info(
			fmt.Sprintf("did not find the cronjobs api in %s", strings.Join(apiGroupVersions, " or ")),
		)
	} else {
		viper.Set(v1.FlagCronJobsVersion, detectedVersion)
		log.Log.V(-1).Info(fmt.Sprintf("found the cronjobs api in %s", detectedVersion))
	}
}

func (b *Background) detectAutoscalingVersion(ctx context.Context) {
	apiGroupVersions := []string{v1.FlagAutoscalingVersionV2, v1.FlagAutoscalingVersionV2Beta2}
	detectedVersion := ""

	for _, apiGroupVersion := range apiGroupVersions {
		groupAPIList, err := b.dcl.ServerResourcesForGroupVersion(apiGroupVersion)
		if err != nil {
			log.Log.V(-1).Info(
				fmt.Sprintf("error getting %s api list: %s", apiGroupVersion, err),
			)
			continue
		}
		for _, api := range groupAPIList.APIResources {
			if api.Name == "horizontalpodautoscalers" {
				detectedVersion = apiGroupVersion
				break
			}
		}
	}

	if detectedVersion == "" {
		log.Log.V(2).Info(
			fmt.Sprintf("did not find the autoscaling api in %s", strings.Join(apiGroupVersions, " or ")),
		)
	} else {
		viper.Set(v1.FlagAutoscalingVersion, detectedVersion)
		log.Log.V(-1).Info(fmt.Sprintf("found the horizontalpodautoscalers api in %s", detectedVersion))
	}
}

// AvailableAPIs returns available list of CRDs from the cluster.
func AvailableAPIs(discovery discovery.DiscoveryInterface, groups map[string]bool) ([]*metav1.APIResourceList, error) {
	var apiLists []*metav1.APIResourceList
	groupList, err := discovery.ServerGroups()
	if err != nil {
		return apiLists, err
	}

	var errors error
	for _, sg := range groupList.Groups {
		if groups[sg.Name] {
			groupAPIList, err := discovery.ServerResourcesForGroupVersion(sg.PreferredVersion.GroupVersion)
			if err == nil {
				apiLists = append(apiLists, groupAPIList)
			} else {
				errors = fmt.Errorf("%w; Error getting resources for server group %s: %w", errors, sg.Name, err)
			}
		}
	}
	return apiLists, errors
}

func (b *Background) detectPlatform(ctx context.Context, apiList []*metav1.APIResourceList) {
	// detect the platform, we run this only once, as the platform can't change between runs ;)
	platform := OperatorConfiguration.GetPlatform()
	detectedPlatform := ""

	if !OperatorConfiguration.IsPlatformAutodetectionEnabled() {
		log.Log.V(-1).Info(
			"The 'platform' option is explicitly set",
			"platform", platform,
		)
		return
	}

	log.Log.V(-1).Info("Attempting to auto-detect the platform")
	if isOpenShift(apiList) {
		detectedPlatform = OpenShiftPlatform.String()
	} else {
		detectedPlatform = KubernetesPlatform.String()
	}

	OperatorConfiguration.SetPlatform(detectedPlatform)
	log.Log.Info(
		"Auto-detected the platform",
		"platform", detectedPlatform,
	)
}

func (b *Background) detectOAuthProxyImageStream(ctx context.Context) {
	tracer := otel.GetTracerProvider().Tracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "detectOAuthProxyImageStream")
	defer span.End()

	if OperatorConfiguration.GetPlatform() != OpenShiftPlatform {
		log.Log.V(-1).Info(
			"Not running on OpenShift, so won't configure OAuthProxy imagestream.",
		)
		return
	}

	imageStreamNamespace := viper.GetString("openshift-oauth-proxy-imagestream-ns")
	imageStreamName := viper.GetString("openshift-oauth-proxy-imagestream-name")
	if imageStreamNamespace == "" || imageStreamName == "" {
		log.Log.Info(
			"OAuthProxy ImageStream namespace and/or name not defined",
			"namespace", imageStreamNamespace,
			"name", imageStreamName,
		)
		return
	}

	// if the image is already digest-based no need to get the reference from an ImageStream
	currImage := OperatorConfiguration.GetOautProxyImage()
	currImageReference, err := imagereference.Parse(currImage)
	if err == nil {
		if currImageReference.ID != "" {
			log.Log.V(6).Info(
				"OAuthProxy Image already digest-based",
				"namespace", imageStreamNamespace,
				"name", imageStreamName,
			)
			return
		}
	}

	imageStream := &osimagev1.ImageStream{}
	namespacedName := types.NamespacedName{
		Name:      imageStreamName,
		Namespace: imageStreamNamespace,
	}

	if err = b.cl.Get(ctx, namespacedName, imageStream); err != nil {
		log.Log.Error(
			err,
			"Failed to obtain OAuthProxy ImageStream",
			"namespace", imageStreamNamespace,
			"name", imageStreamName,
		)
		tracing.HandleError(err, span)
		return
	}

	if len(imageStream.Status.Tags) == 0 {
		log.Log.V(6).Info(
			"OAuthProxy ImageStream has no tags",
			"namespace", imageStreamNamespace,
			"name", imageStreamName,
		)
		return
	}

	if len(imageStream.Status.Tags[0].Items) == 0 {
		log.Log.V(6).Info(
			"OAuthProxy ImageStream tag has no items",
			"namespace", imageStreamNamespace,
			"name", imageStreamName,
		)
		return
	}

	if len(imageStream.Status.Tags[0].Items[0].DockerImageReference) == 0 {
		log.Log.V(5).Info(
			"OAuthProxy ImageStream tag has no DockerImageReference",
			"namespace", imageStreamNamespace,
			"name", imageStreamName,
		)
		return
	}

	image := imageStream.Status.Tags[0].Items[0].DockerImageReference

	OperatorConfiguration.SetOautProxyImage(image)
	log.Log.Info(
		"Updated OAuth Proxy image flag",
		"image", image,
	)
}

func (b *Background) detectElasticsearch(ctx context.Context, apiList []*metav1.APIResourceList) {
	// detect whether the Elasticsearch operator is available
	currentESProvision := OperatorConfiguration.GetESPIntegration()
	if !b.retryDetectEs {
		log.Log.V(-1).Info(
			"ES Operator integration explicitly set",
			v1.FlagESProvision, currentESProvision.String(),
		)
	}

	log.Log.V(-1).Info("Determining whether we should enable the Elasticsearch Operator integration")

	esProvision := ESOperatorIntegrationNo
	if IsElasticsearchOperatorAvailable(apiList) {
		esProvision = ESOperatorIntegrationYes
	}

	if currentESProvision != esProvision {
		log.Log.Info(
			"Automatically adjusted the integration with the ES Operator",
			v1.FlagESProvision, esProvision.String(),
		)
		OperatorConfiguration.SetESIngration(esProvision)
	}
}

// detectKafka checks whether the Kafka Operator is available
func (b *Background) detectKafka(_ context.Context, apiList []*metav1.APIResourceList) {
	currentKafkaProvision := OperatorConfiguration.GetKafkaIntegration()
	if !b.retryDetectKafka {
		log.Log.V(-1).Info(
			"The 'kafka-provision' option is explicitly set",
			"kafka-provision", currentKafkaProvision.String(),
		)
		return
	}

	log.Log.V(-1).Info("Determining whether we should enable the Kafka Operator integration")

	kafkaProvision := KafkaOperatorIntegrationNo
	if isKafkaOperatorAvailable(apiList) {
		kafkaProvision = KafkaOperatorIntegrationYes
	}

	if currentKafkaProvision != kafkaProvision {
		log.Log.Info(
			"Automatically adjusted the 'kafka-provision' flag",
			"kafka-provision", kafkaProvision.String(),
		)
		OperatorConfiguration.SetKafkaIntegration(kafkaProvision)
	}
}

func (b *Background) detectClusterRoles(ctx context.Context) {
	if OperatorConfiguration.GetPlatform() != OpenShiftPlatform {
		return
	}
	tr := &authenticationapi.TokenReview{
		ObjectMeta: metav1.ObjectMeta{Name: "jaeger-operator-TEST"},
		Spec: authenticationapi.TokenReviewSpec{
			Token: "TEST",
		},
	}
	currentAuthDelegator := OperatorConfiguration.GetAuthDelegator()
	var newAuthDelegator AuthDelegatorAvailability
	if err := b.cl.Create(ctx, tr); err != nil {
		if !OperatorConfiguration.IsAuthDelegatorSet() || OperatorConfiguration.IsAuthDelegatorAvailable() {
			// for the first run, we log this info, or when the previous value was true
			log.Log.Info(
				"The service account running this operator does not have the role 'system:auth-delegator', consider granting it for additional capabilities",
			)
		}
		newAuthDelegator = AuthDelegatorAvailabilityNo
	} else {
		// this isn't technically correct, as we only ensured that we can create token reviews (which is what the OAuth Proxy does)
		// but it might be the case that we have *another* cluster role that includes this access and still not have
		// the "system:auth-delegator". This is an edge case, and it's more complicated to check that, so, we'll keep it simple for now
		// and deal with the edge case if it ever manifests in the real world
		if !OperatorConfiguration.IsAuthDelegatorSet() || (OperatorConfiguration.IsAuthDelegatorSet() && !OperatorConfiguration.IsAuthDelegatorAvailable()) {
			// for the first run, we log this info, or when the previous value was 'false'
			log.Log.Info(
				"The service account running this operator has the role 'system:auth-delegator', enabling OAuth Proxy's 'delegate-urls' option",
			)
		}
		newAuthDelegator = AuthDelegatorAvailabilityYes
	}

	if currentAuthDelegator != newAuthDelegator || !OperatorConfiguration.IsAuthDelegatorSet() {
		OperatorConfiguration.SetAuthDelegatorAvailability(newAuthDelegator)
	}

	if err := b.cl.Delete(ctx, tr); err != nil {
		// Remove the test Token.
		// If the token could not be created due to permissions, we're ok.
		// If the token was created, we remove it to ensure the next iteration doesn't fail.
		// If the token creation failed because it was created before, we remove it to ensure the next iteration doesn't fail.
		log.Log.V(2).Info("The jaeger-operator-TEST TokenReview could not be removed: %w", err)
	}
}

func isOpenShift(apiList []*metav1.APIResourceList) bool {
	for _, r := range apiList {
		if strings.HasPrefix(r.GroupVersion, "route.openshift.io") {
			return true
		}
	}
	return false
}

// IsElasticsearchOperatorAvailable returns true if OpenShift Elasticsearch CRD is available in the cluster.
func IsElasticsearchOperatorAvailable(apiList []*metav1.APIResourceList) bool {
	for _, r := range apiList {
		if strings.HasPrefix(r.GroupVersion, "logging.openshift.io") {
			for _, api := range r.APIResources {
				if api.Kind == "Elasticsearch" {
					return true
				}
			}
		}
	}
	return false
}

func isKafkaOperatorAvailable(apiList []*metav1.APIResourceList) bool {
	for _, r := range apiList {
		if strings.HasPrefix(r.GroupVersion, "kafka.strimzi.io") {
			return true
		}
	}
	return false
}
