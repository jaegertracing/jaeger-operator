package autodetect

import (
	"context"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	authenticationapi "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/jaegertracing/jaeger-operator/pkg/ingress"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

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

	done := make(chan bool)
	go func() {
		b.autoDetectCapabilities()
		done <- true
	}()

	go func() {
		for {
			select {
			case <-done:
				log.Trace("finished the first auto-detection")
			case <-b.ticker.C:
				b.autoDetectCapabilities()
			}
		}
	}()
}

// Stop causes the background process to stop auto detecting capabilities
func (b *Background) Stop() {
	b.ticker.Stop()
}

func (b *Background) autoDetectCapabilities() {
	ctx := context.Background()

	apiList, err := b.availableAPIs(ctx)
	if err != nil {
		log.WithError(err).Info("failed to determine the platform capabilities, auto-detected properties will remain the same until next cycle.")
	} else {

		b.firstRun.Do(func() {
			// the platform won't change during the execution of the operator, need to run it only once
			b.detectPlatform(ctx, apiList)
			b.detectIngressAPI()

		})

		b.detectElasticsearch(ctx, apiList)
		b.detectKafka(ctx, apiList)
	}

	b.detectClusterRoles(ctx)
	b.cleanDeployments(ctx)
}

func (b *Background) availableAPIs(ctx context.Context) (*metav1.APIGroupList, error) {
	apiList, err := b.dcl.ServerGroups()
	if err != nil {
		return nil, err
	}

	return apiList, nil
}

func (b *Background) detectPlatform(ctx context.Context, apiList *metav1.APIGroupList) {
	// detect the platform, we run this only once, as the platform can't change between runs ;)
	if strings.EqualFold(viper.GetString("platform"), v1.FlagPlatformAutoDetect) {
		log.Trace("Attempting to auto-detect the platform")
		if isOpenShift(apiList) {
			viper.Set("platform", v1.FlagPlatformOpenShift)
		} else {
			viper.Set("platform", v1.FlagPlatformKubernetes)
		}

		log.WithField("platform", viper.GetString("platform")).Info("Auto-detected the platform")
	} else {
		log.WithField("platform", viper.GetString("platform")).Debug("The 'platform' option is explicitly set")
	}
}

func (b *Background) detectIngressAPI() {
	apiRes, err := b.dcl.ServerResourcesForGroupVersion("networking.k8s.io/v1beta1")
	if err != nil {
		viper.Set("ingress-api", ingress.ExtensionAPI)
		log.WithField("ingress-api", viper.GetString("ingress-api")).Info("Auto-detected ingress api")
		return
	}

	for _, r := range apiRes.APIResources {
		if r.Name == "ingresses" {
			viper.Set("ingress-api", ingress.NetworkingAPI)
			log.WithField("ingress-api", viper.GetString("ingress-api")).Info("Auto-detected ingress api")
			break
		}
	}
}

func (b *Background) detectElasticsearch(ctx context.Context, apiList *metav1.APIGroupList) {
	// detect whether the Elasticsearch operator is available
	if b.retryDetectEs {
		log.Trace("Determining whether we should enable the Elasticsearch Operator integration")
		previous := viper.GetString("es-provision")
		if isElasticsearchOperatorAvailable(apiList) {
			viper.Set("es-provision", v1.FlagProvisionElasticsearchYes)
		} else {
			viper.Set("es-provision", v1.FlagProvisionElasticsearchNo)
		}

		if previous != viper.GetString("es-provision") {
			log.WithField("es-provision", viper.GetString("es-provision")).Info("Automatically adjusted the 'es-provision' flag")
		}
	} else {
		log.WithField("es-provision", viper.GetString("es-provision")).Trace("The 'es-provision' option is explicitly set")
	}
}

// detectKafka checks whether the Kafka Operator is available
func (b *Background) detectKafka(ctx context.Context, apiList *metav1.APIGroupList) {
	// viper has a "IsSet" method that we could use, except that it returns "true" even
	// when nothing is set but it finds a 'Default' value...
	if b.retryDetectKafka {
		log.Trace("Determining whether we should enable the Kafka Operator integration")

		previous := viper.GetString("kafka-provision")
		if isKafkaOperatorAvailable(apiList) {
			viper.Set("kafka-provision", v1.FlagProvisionKafkaYes)
		} else {
			viper.Set("kafka-provision", v1.FlagProvisionKafkaNo)
		}

		if previous != viper.GetString("kafka-provision") {
			log.WithField("kafka-provision", viper.GetString("kafka-provision")).Info("Automatically adjusted the 'kafka-provision' flag")
		}
	} else {
		log.WithField("kafka-provision", viper.GetString("kafka-provision")).Trace("The 'kafka-provision' option is explicitly set")
	}
}

func (b *Background) detectClusterRoles(ctx context.Context) {
	tr := &authenticationapi.TokenReview{
		ObjectMeta: metav1.ObjectMeta{Name: "jaeger-operator-TEST"},
		Spec: authenticationapi.TokenReviewSpec{
			Token: "TEST",
		},
	}
	if err := b.cl.Create(ctx, tr); err != nil {
		if !viper.IsSet("auth-delegator-available") || (viper.IsSet("auth-delegator-available") && viper.GetBool("auth-delegator-available")) {
			// for the first run, we log this info, or when the previous value was true
			log.Info("The service account running this operator does not have the role 'system:auth-delegator', consider granting it for additional capabilities")
		}
		viper.Set("auth-delegator-available", false)
	} else {
		// this isn't technically correct, as we only ensured that we can create token reviews (which is what the OAuth Proxy does)
		// but it might be the case that we have *another* cluster role that includes this access and still not have
		// the "system:auth-delegator". This is an edge case, and it's more complicated to check that, so, we'll keep it simple for now
		// and deal with the edge case if it ever manifests in the real world
		if !viper.IsSet("auth-delegator-available") || (viper.IsSet("auth-delegator-available") && !viper.GetBool("auth-delegator-available")) {
			// for the first run, we log this info, or when the previous value was 'false'
			log.Info("The service account running this operator has the role 'system:auth-delegator', enabling OAuth Proxy's 'delegate-urls' option")
		}
		viper.Set("auth-delegator-available", true)
	}
}

func (b *Background) cleanDeployments(ctx context.Context) {
	log.Trace("detecting orphaned deployments.")

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
				log.WithField("namespace", ns).WithError(err).Error("error getting a list of deployments to analyze in namespace")
			}
			deployments.Items = append(deployments.Items, nsDeps.Items...)

			instances := &v1.JaegerList{}
			if err := b.clReader.List(ctx, instances, client.InNamespace(ns)); err != nil {
				log.WithField("namespace", ns).WithError(err).Error("error getting a list of existing jaeger instances in namespace")
			}
			for _, jaeger := range instances.Items {
				instancesMap[jaeger.Name] = &jaeger
			}
		}
	} else {
		if err := b.clReader.List(ctx, deployments, deployOpts...); err != nil {
			log.WithError(err).Error("error getting a list of deployments to analyze")
		}

		instances := &v1.JaegerList{}
		if err := b.clReader.List(ctx, instances); err != nil {
			log.WithError(err).Error("error getting a list of existing jaeger instances")
		}
		for _, jaeger := range instances.Items {
			instancesMap[jaeger.Name] = &jaeger
		}
	}

	// check deployments to see which one needs to be cleaned.
	for _, dep := range deployments.Items {
		if instanceName, ok := dep.Labels[inject.Label]; ok {
			_, instanceExists := instancesMap[instanceName]
			if !instanceExists { // Jaeger instance not exist anymore, we need to clean this up.
				inject.CleanSidecar(instanceName, &dep)
				if err := b.cl.Update(ctx, &dep); err != nil {
					log.WithFields(log.Fields{
						"deploymentName":      dep.Name,
						"deploymentNamespace": dep.Namespace,
					}).WithError(err).Error("error cleaning orphaned deployment")
				}
			}
		}
	}
}

func isOpenShift(apiList *metav1.APIGroupList) bool {
	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "route.openshift.io" {
			return true
		}
	}
	return false
}

func isElasticsearchOperatorAvailable(apiList *metav1.APIGroupList) bool {
	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "logging.openshift.io" {
			return true
		}
	}
	return false
}

func isKafkaOperatorAvailable(apiList *metav1.APIGroupList) bool {
	apiGroups := apiList.Groups
	for i := 0; i < len(apiGroups); i++ {
		if apiGroups[i].Name == "kafka.strimzi.io" {
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
			log.Warnf("failed to build label selector: %v", err)
			return
		}
		sel.Add(*req)
	}
	opts.LabelSelector = sel
}
