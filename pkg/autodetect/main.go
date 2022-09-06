package autodetect

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	authenticationapi "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
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
		})

		b.detectElasticsearch(ctx, apiList)
		b.detectKafka(ctx, apiList)
		b.detectCronjobsVersion(ctx)
		b.detectAutoscalingVersion(ctx)
	}

	b.detectClusterRoles(ctx)
	b.cleanDeployments(ctx)
}

func (b *Background) detectCronjobsVersion(ctx context.Context) {
	apiGroupVersions := []string{v1.FlagCronJobsVersionBatchV1, v1.FlagCronJobsVersionBatchV1Beta1}
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
				viper.Set(v1.FlagCronJobsVersion, apiGroupVersion)
				log.Log.V(-1).Info(fmt.Sprintf("found the cronjobs api in %s", apiGroupVersion))
				return
			}
		}
	}

	log.Log.V(2).Info(
		fmt.Sprintf("did not find the cronjobs api in %s", strings.Join(apiGroupVersions, " or ")),
	)
}

func (b *Background) detectAutoscalingVersion(ctx context.Context) {
	apiGroupVersions := []string{v1.FlagAutoscalingVersionV2, v1.FlagAutoscalingVersionV2Beta2}
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
				viper.Set(v1.FlagAutoscalingVersion, apiGroupVersion)
				log.Log.V(-1).Info(fmt.Sprintf("found the horizontalpodautoscalers api in %s", apiGroupVersion))
				return
			}
		}
	}

	log.Log.V(2).Info(
		fmt.Sprintf("did not find the autoscaling api in %s", strings.Join(apiGroupVersions, " or ")),
	)
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
				errors = fmt.Errorf("%v; Error getting resources for server group %s: %v", errors, sg.Name, err)
			}
		}
	}
	return apiLists, errors
}

func (b *Background) detectPlatform(ctx context.Context, apiList []*metav1.APIResourceList) {
	// detect the platform, we run this only once, as the platform can't change between runs ;)
	if strings.EqualFold(viper.GetString("platform"), v1.FlagPlatformAutoDetect) {
		log.Log.V(-1).Info("Attempting to auto-detect the platform")
		if isOpenShift(apiList) {
			viper.Set("platform", v1.FlagPlatformOpenShift)
		} else {
			viper.Set("platform", v1.FlagPlatformKubernetes)
		}

		log.Log.Info(
			"Auto-detected the platform",
			"platform", viper.GetString("platform"),
		)
	} else {
		log.Log.V(-1).Info(
			"The 'platform' option is explicitly set",
			"platform", viper.GetString("platform"),
		)
	}
}

func (b *Background) detectElasticsearch(ctx context.Context, apiList []*metav1.APIResourceList) {
	// detect whether the Elasticsearch operator is available
	if b.retryDetectEs {
		log.Log.V(-1).Info(
			"Determining whether we should enable the Elasticsearch Operator integration",
		)
		previous := viper.GetString("es-provision")
		if IsElasticsearchOperatorAvailable(apiList) {
			viper.Set("es-provision", v1.FlagProvisionElasticsearchYes)
		} else {
			viper.Set("es-provision", v1.FlagProvisionElasticsearchNo)
		}

		if previous != viper.GetString("es-provision") {
			log.Log.Info(
				"Automatically adjusted the 'es-provision' flag",
				"es-provision", viper.GetString("es-provision"),
			)
		}
	} else {
		log.Log.V(-1).Info(
			"The 'es-provision' option is explicitly set",
			"es-provision", viper.GetString("es-provision"),
		)
	}
}

// detectKafka checks whether the Kafka Operator is available
func (b *Background) detectKafka(_ context.Context, apiList []*metav1.APIResourceList) {
	// viper has a "IsSet" method that we could use, except that it returns "true" even
	// when nothing is set but it finds a 'Default' value...
	if b.retryDetectKafka {
		log.Log.V(-1).Info("Determining whether we should enable the Kafka Operator integration")

		previous := viper.GetString("kafka-provision")
		if isKafkaOperatorAvailable(apiList) {
			viper.Set("kafka-provision", v1.FlagProvisionKafkaYes)
		} else {
			viper.Set("kafka-provision", v1.FlagProvisionKafkaNo)
		}

		if previous != viper.GetString("kafka-provision") {
			log.Log.Info(
				"Automatically adjusted the 'kafka-provision' flag",
				"kafka-provision", viper.GetString("kafka-provision"),
			)
		}
	} else {
		log.Log.V(-1).Info(
			"The 'kafka-provision' option is explicitly set",
			"kafka-provision", viper.GetString("kafka-provision"),
		)
	}
}

func (b *Background) detectClusterRoles(ctx context.Context) {
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return
	}
	tr := &authenticationapi.TokenReview{
		ObjectMeta: metav1.ObjectMeta{Name: "jaeger-operator-TEST"},
		Spec: authenticationapi.TokenReviewSpec{
			Token: "TEST",
		},
	}
	if err := b.cl.Create(ctx, tr); err != nil {
		if !viper.IsSet("auth-delegator-available") || (viper.IsSet("auth-delegator-available") && viper.GetBool("auth-delegator-available")) {
			// for the first run, we log this info, or when the previous value was true
			log.Log.Info(
				"The service account running this operator does not have the role 'system:auth-delegator', consider granting it for additional capabilities",
			)
		}
		viper.Set("auth-delegator-available", false)
	} else {
		// this isn't technically correct, as we only ensured that we can create token reviews (which is what the OAuth Proxy does)
		// but it might be the case that we have *another* cluster role that includes this access and still not have
		// the "system:auth-delegator". This is an edge case, and it's more complicated to check that, so, we'll keep it simple for now
		// and deal with the edge case if it ever manifests in the real world
		if !viper.IsSet("auth-delegator-available") || (viper.IsSet("auth-delegator-available") && !viper.GetBool("auth-delegator-available")) {
			// for the first run, we log this info, or when the previous value was 'false'
			log.Log.Info(
				"The service account running this operator has the role 'system:auth-delegator', enabling OAuth Proxy's 'delegate-urls' option",
			)
		}
		viper.Set("auth-delegator-available", true)
	}
}

func (b *Background) cleanDeployments(ctx context.Context) {
	log.Log.V(-1).Info("detecting orphaned deployments.")

	instancesMap := make(map[string]*v1.Jaeger)
	deployments := &appsv1.DeploymentList{}
	deployOpts := []client.ListOption{
		matchingLabelKeys(map[string]string{inject.Label: ""}),
	}

	// if we are not watching all namespaces, we have to get items from each namespace being watched
	if namespaces := viper.GetString(v1.ConfigWatchNamespace); namespaces != v1.WatchAllNamespaces {
		for _, ns := range strings.Split(namespaces, ",") {
			nsDeps := &appsv1.DeploymentList{}
			if err := b.clReader.List(ctx, nsDeps, append(deployOpts, client.InNamespace(ns))...); err != nil {
				log.Log.Error(
					err,
					"error getting a list of deployments to analyze in namespace",
					"namespace", ns,
				)
			}
			deployments.Items = append(deployments.Items, nsDeps.Items...)

			instances := &v1.JaegerList{}
			if err := b.clReader.List(ctx, instances, client.InNamespace(ns)); err != nil {
				log.Log.Error(
					err,
					"error getting a list of existing jaeger instances in namespace",
					"namespace", ns,
				)
			}
			for i := range instances.Items {
				instancesMap[instances.Items[i].Name] = &instances.Items[i]
			}
		}
	} else {
		if err := b.clReader.List(ctx, deployments, deployOpts...); err != nil {
			log.Log.Error(
				err,
				"error getting a list of deployments to analyze",
			)
		}

		instances := &v1.JaegerList{}
		if err := b.clReader.List(ctx, instances); err != nil {
			log.Log.Error(
				err,
				"error getting a list of existing jaeger instances",
			)
		}
		for i := range instances.Items {
			instancesMap[instances.Items[i].Name] = &instances.Items[i]
		}
	}

	// check deployments to see which one needs to be cleaned.
	for i := range deployments.Items {
		dep := deployments.Items[i]
		if instanceName, ok := dep.Labels[inject.Label]; ok {
			_, instanceExists := instancesMap[instanceName]
			if !instanceExists { // Jaeger instance not exist anymore, we need to clean this up.
				inject.CleanSidecar(instanceName, &dep)
				if err := b.cl.Update(ctx, &dep); err != nil {
					log.Log.Error(
						err,
						"error cleaning orphaned deployment",
						"deploymentName", dep.Name,
						"deploymentNamespace", dep.Namespace,
					)
				}
			}
		}
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

type matchingLabelKeys map[string]string

func (m matchingLabelKeys) ApplyToList(opts *client.ListOptions) {
	sel := labels.NewSelector()
	for k := range map[string]string(m) {
		req, err := labels.NewRequirement(k, selection.Exists, []string{})
		if err != nil {
			log.Log.Error(err, "failed to build label selector")
			return
		}
		sel.Add(*req)
	}
	opts.LabelSelector = sel
}
