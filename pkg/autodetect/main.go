package autodetect

import (
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	authenticationapi "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// Background represents a procedure that runs in the background, periodically auto-detecting features
type Background struct {
	cl     client.Client
	dcl    discovery.DiscoveryInterface
	ticker *time.Ticker
}

// New creates a new auto-detect runner
func New(mgr manager.Manager) (*Background, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	return &Background{dcl: dcl, cl: mgr.GetClient()}, nil
}

// WithClients builds a new Background with the provided clients
func WithClients(cl client.Client, dcl discovery.DiscoveryInterface) *Background {
	return &Background{cl: cl, dcl: dcl}
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
				log.Debug("finished the first auto-detection")
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
	apiList, err := b.availableAPIs()
	if err != nil {
		log.WithError(err).Info("Failed to determine the platform capabilities. Auto-detected properties will fallback to their default values.")
		viper.Set("platform", v1.FlagPlatformKubernetes)
		viper.Set("es-provision", v1.FlagProvisionElasticsearchFalse)
	} else {
		// we could run all the detect* functions in parallel, but let's keep it simple for now
		b.detectPlatform(apiList)
		b.detectElasticsearch(apiList)
	}

	b.detectClusterRoles()
}

func (b *Background) availableAPIs() (*metav1.APIGroupList, error) {
	apiList, err := b.dcl.ServerGroups()
	if err != nil {
		return nil, err
	}

	return apiList, nil
}

func (b *Background) detectPlatform(apiList *metav1.APIGroupList) {
	// detect the platform
	if strings.EqualFold(viper.GetString("platform"), v1.FlagPlatformAutoDetect) {
		log.Debug("Attempting to auto-detect the platform")
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

func (b *Background) detectElasticsearch(apiList *metav1.APIGroupList) {
	// detect whether the Elasticsearch operator is available
	if strings.EqualFold(viper.GetString("es-provision"), v1.FlagProvisionElasticsearchAuto) {
		log.Debug("Determining whether we should enable the Elasticsearch Operator integration")
		if isElasticsearchOperatorAvailable(apiList) {
			viper.Set("es-provision", v1.FlagProvisionElasticsearchTrue)
		} else {
			viper.Set("es-provision", v1.FlagProvisionElasticsearchFalse)
		}

		log.WithField("es-provision", viper.GetString("es-provision")).Info("Automatically adjusted the 'es-provision' flag")
	} else {
		log.WithField("es-provision", viper.GetString("es-provision")).Debug("The 'es-provision' option is explicitly set")
	}
}

func (b *Background) detectClusterRoles() {
	tr := &authenticationapi.TokenReview{
		Spec: authenticationapi.TokenReviewSpec{
			Token: "TEST",
		},
	}
	if err := b.cl.Create(context.Background(), tr); err != nil {
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
