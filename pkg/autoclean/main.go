package autoclean

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/viper"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

type Background struct {
	cl       client.Client
	clReader client.Reader
	dcl      discovery.DiscoveryInterface
	ticker   *time.Ticker
}

// New creates a new auto-clean runner
func New(mgr manager.Manager) (*Background, error) {
	dcl, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	return WithClients(mgr.GetClient(), dcl, mgr.GetAPIReader()), nil
}

// WithClients builds a new Background with the provided clients
func WithClients(cl client.Client, dcl discovery.DiscoveryInterface, clr client.Reader) *Background {
	return &Background{
		cl:       cl,
		dcl:      dcl,
		clReader: clr,
	}
}

// Start initializes the auto-clean process that runs in the background
func (b *Background) Start() {
	b.ticker = time.NewTicker(5 * time.Second)
	b.autoClean()

	go func() {
		for {
			<-b.ticker.C
			b.autoClean()
		}
	}()
}

// Stop causes the background process to stop auto clean capabilities
func (b *Background) Stop() {
	b.ticker.Stop()
}

func (b *Background) autoClean() {
	ctx := context.Background()
	b.cleanDeployments(ctx)
}

func (b *Background) cleanDeployments(ctx context.Context) {
	log.Log.V(-1).Info("cleaning orphaned deployments.")

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
