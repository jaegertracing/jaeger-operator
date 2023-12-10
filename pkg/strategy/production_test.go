package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/consolelink"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func init() {
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
	viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
}

func TestCreateProductionDeployment(t *testing.T) {
	name := "TestCreateProductionDeployment"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	c := newProductionStrategy(context.Background(), jaeger)
	assertDeploymentsAndServicesForProduction(t, jaeger, c, false, false, false)
}

func TestCreateProductionDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateProductionDeploymentOnOpenShift"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	normalize(context.Background(), jaeger)

	c := newProductionStrategy(context.Background(), jaeger)
	assertDeploymentsAndServicesForProduction(t, jaeger, c, false, true, false)
}

func TestCreateProductionDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateProductionDeploymentWithDaemonSetAgent"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newProductionStrategy(context.Background(), j)
	assertDeploymentsAndServicesForProduction(t, j, c, true, false, false)
}

func TestCreateProductionDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateProductionDeploymentWithUIConfigMap"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.UI.Options = v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newProductionStrategy(context.Background(), j)
	assertDeploymentsAndServicesForProduction(t, j, c, false, false, true)
}

func TestOptionsArePassed(t *testing.T) {
	jaeger := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-prod",
			Namespace: "simple-prod-ns",
		},
		Spec: v1.JaegerSpec{
			Strategy: v1.DeploymentStrategyProduction,
			Storage: v1.JaegerStorageSpec{
				Type: v1.JaegerESStorage,
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://elasticsearch.default.svc:9200",
					"es.username":    "elastic",
					"es.password":    "changeme",
				}),
			},
		},
	}

	ctrl := For(context.Background(), jaeger)
	deployments := ctrl.Deployments()
	for _, dep := range deployments {
		args := dep.Spec.Template.Spec.Containers[0].Args
		if strings.Contains(dep.Name, "collector") {
			// Including parameter for sampling config
			assert.Len(t, args, 4)
		} else {
			assert.Len(t, args, 3)
		}
		var escount int
		for _, arg := range args {
			if strings.Contains(arg, "es.") {
				escount++
			}
		}
		assert.Equal(t, 3, escount)
	}
}

func TestDelegateProductionDependencies(t *testing.T) {
	// for now, we just have storage dependencies
	j := v1.NewJaeger(types.NamespacedName{Name: "TestDelegateProductionDependencies"})
	j.Spec.Storage.Type = v1.JaegerCassandraStorage
	c := newProductionStrategy(context.Background(), j)
	assert.Equal(t, c.Dependencies(), storage.Dependencies(j))
}

func TestAutoscaleForProduction(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := newProductionStrategy(context.Background(), j)
	assert.Len(t, c.HorizontalPodAutoscalers(), 1)
}

func assertDeploymentsAndServicesForProduction(t *testing.T, instance *v1.Jaeger, s S, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
	name := instance.Name

	deployments := map[string]bool{
		fmt.Sprintf("%s-collector", name): false,
		fmt.Sprintf("%s-query", name):     false,
	}

	daemonsets := map[string]bool{
		fmt.Sprintf("%s-agent-daemonset", name): !hasDaemonSet,
	}

	services := map[string]bool{
		fmt.Sprintf("%s-collector", strings.ToLower(name)): false,
		fmt.Sprintf("%s-query", strings.ToLower(name)):     false,
	}

	ingresses := map[string]bool{}
	routes := map[string]bool{}
	consoleLinks := map[string]bool{}
	if autodetect.OperatorConfiguration.GetPlatform() == autodetect.OpenShiftPlatform {
		routes[util.DNSName(name)] = false
		consoleLinks[consolelink.Name(instance)] = false
	} else {
		ingresses[fmt.Sprintf("%s-query", name)] = false
	}

	serviceAccounts := map[string]bool{name: false}
	if hasOAuthProxy {
		serviceAccounts[fmt.Sprintf("%s-ui-proxy", name)] = false
	}

	configMaps := map[string]bool{}
	if hasConfigMap {
		configMaps[fmt.Sprintf("%s-ui-configuration", name)] = false
	}
	assertHasAllObjects(t, name, s, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps, consoleLinks)
}

func TestSparkDependenciesProduction(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1.Jaeger) S {
		return newProductionStrategy(context.Background(), jaeger)
	})
}

func TestEsIndexCleanerProduction(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1.Jaeger) S {
		return newProductionStrategy(context.Background(), jaeger)
	})
}

func TestAgentSidecarIsInjectedIntoQueryForStreamingForProduction(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "TestAgentSidecarIsInjectedIntoQueryForStreamingForProduction"})
	c := newProductionStrategy(context.Background(), j)
	for _, dep := range c.Deployments() {
		if strings.HasSuffix(dep.Name, "-query") {
			assert.Equal(t, "TestAgentSidecarIsInjectedIntoQueryForStreamingForProduction", dep.Annotations["sidecar.jaegertracing.io/inject"])
			assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
			assert.Equal(t, "jaeger-query", dep.Spec.Template.Spec.Containers[0].Name)
		}
	}
}

func TestAgentSidecarNotInjectedTracingEnabledFalseForProduction(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "TestAgentSidecarNotInjectedTracingEnabledFalseForProduction"})
	falseVar := false
	j.Spec.Query.TracingEnabled = &falseVar
	c := newProductionStrategy(context.Background(), j)
	for _, dep := range c.Deployments() {
		if strings.HasSuffix(dep.Name, "-query") {
			assert.Len(t, dep.Spec.Template.Spec.Containers, 1)
		}
	}
}

func TestElasticsearchInject(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: t.Name()})
	j.Spec.Storage.Type = v1.JaegerESStorage
	verdad := true
	one := int(1)
	j.Spec.Storage.EsIndexCleaner.Enabled = &verdad
	j.Spec.Storage.EsIndexCleaner.NumberOfDays = &one
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.use-aliases": true})
	c := newProductionStrategy(context.Background(), j)
	// there should be index-cleaner, rollover, lookback
	assert.Len(t, c.cronJobs, 3)
	assertEsInjectSecrets(t, c.cronJobs[0].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec)
	assertEsInjectSecrets(t, c.cronJobs[1].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec)
	assertEsInjectSecrets(t, c.cronJobs[2].(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec)
}

func assertEsInjectSecrets(t *testing.T, p corev1.PodSpec) {
	assert.Len(t, p.Volumes, 1)
	assert.Equal(t, "certs", p.Volumes[0].Name)
	assert.Equal(t, "certs", p.Containers[0].VolumeMounts[0].Name)
	envs := map[string]corev1.EnvVar{}
	for _, e := range p.Containers[0].Env {
		envs[e.Name] = e
	}
	assert.Contains(t, envs, "ES_TLS_ENABLED")
	assert.Contains(t, envs, "ES_TLS_CA")
	assert.Contains(t, envs, "ES_TLS_KEY")
	assert.Contains(t, envs, "ES_TLS_CERT")
}
