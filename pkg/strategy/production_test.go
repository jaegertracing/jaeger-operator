package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateProductionDeployment(t *testing.T) {
	name := "TestCreateProductionDeployment"
	c := newProductionStrategy(v1.NewJaeger(types.NamespacedName{Name: name}), &storage.ElasticsearchDeployment{})
	assertDeploymentsAndServicesForProduction(t, name, c, false, false, false)
}

func TestCreateProductionDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateProductionDeploymentOnOpenShift"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	normalize(jaeger)

	c := newProductionStrategy(jaeger, &storage.ElasticsearchDeployment{})
	assertDeploymentsAndServicesForProduction(t, name, c, false, true, false)
}

func TestCreateProductionDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateProductionDeploymentWithDaemonSetAgent"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newProductionStrategy(j, &storage.ElasticsearchDeployment{})
	assertDeploymentsAndServicesForProduction(t, name, c, true, false, false)
}

func TestCreateProductionDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateProductionDeploymentWithUIConfigMap"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.UI.Options = v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newProductionStrategy(j, &storage.ElasticsearchDeployment{})
	assertDeploymentsAndServicesForProduction(t, name, c, false, false, true)
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
			Strategy: "production",
			Storage: v1.JaegerStorageSpec{
				Type: "elasticsearch",
				Options: v1.NewOptions(map[string]interface{}{
					"es.server-urls": "http://elasticsearch.default.svc:9200",
					"es.username":    "elastic",
					"es.password":    "changeme",
				}),
			},
		},
	}

	ctrl := For(context.TODO(), jaeger, []corev1.Secret{})
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
	j.Spec.Storage.Type = "cassandra"
	c := newProductionStrategy(j, &storage.ElasticsearchDeployment{})
	assert.Equal(t, c.Dependencies(), storage.Dependencies(j))
}

func assertDeploymentsAndServicesForProduction(t *testing.T, name string, s S, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
	expectedNumObjs := 7

	if hasDaemonSet {
		expectedNumObjs++
	}

	if hasOAuthProxy {
		expectedNumObjs++
	}

	if hasConfigMap {
		expectedNumObjs++
	}

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
	if viper.GetString("platform") == v1.FlagPlatformOpenShift {
		routes[name] = false
	} else {
		ingresses[fmt.Sprintf("%s-query", name)] = false
	}

	serviceAccounts := map[string]bool{fmt.Sprintf("%s", name): false}
	if hasOAuthProxy {
		serviceAccounts[fmt.Sprintf("%s-ui-proxy", name)] = false
	}

	configMaps := map[string]bool{}
	if hasConfigMap {
		configMaps[fmt.Sprintf("%s-ui-configuration", name)] = false
	}
	assertHasAllObjects(t, name, s, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps)
}

func TestSparkDependenciesProduction(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1.Jaeger) S {
		return newProductionStrategy(jaeger, &storage.ElasticsearchDeployment{Jaeger: jaeger})
	})
}

func TestEsIndexCleanerProduction(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1.Jaeger) S {
		return newProductionStrategy(jaeger, &storage.ElasticsearchDeployment{Jaeger: jaeger})
	})
}

func TestAgentSidecarIsInjectedIntoQueryForStreamingForProduction(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "TestAgentSidecarIsInjectedIntoQueryForStreamingForProduction"})
	c := newProductionStrategy(j, &storage.ElasticsearchDeployment{})
	for _, dep := range c.Deployments() {
		if strings.HasSuffix(dep.Name, "-query") {
			assert.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))
			assert.Equal(t, "jaeger-agent", dep.Spec.Template.Spec.Containers[1].Name)
		}
	}
}

func TestElasticsearchInject(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: t.Name()})
	j.Spec.Storage.Type = "elasticsearch"
	verdad := true
	one := int(1)
	j.Spec.Storage.EsIndexCleaner.Enabled = &verdad
	j.Spec.Storage.EsIndexCleaner.NumberOfDays = &one
	j.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"es.use-aliases": true})
	es := &storage.ElasticsearchDeployment{Jaeger: j, CertScript: "../../scripts/cert_generation.sh"}
	err := es.CleanCerts()
	require.NoError(t, err)
	defer es.CleanCerts()
	c := newProductionStrategy(j, es)
	// there should be index-cleaner, rollover, lookback
	assert.Equal(t, 3, len(c.cronJobs))
	assertEsInjectSecrets(t, c.cronJobs[0].Spec.JobTemplate.Spec.Template.Spec)
	assertEsInjectSecrets(t, c.cronJobs[1].Spec.JobTemplate.Spec.Template.Spec)
	assertEsInjectSecrets(t, c.cronJobs[2].Spec.JobTemplate.Spec.Template.Spec)
}

func assertEsInjectSecrets(t *testing.T, p corev1.PodSpec) {
	assert.Equal(t, 1, len(p.Volumes))
	assert.Equal(t, "certs", p.Volumes[0].Name)
	assert.Equal(t, "certs", p.Containers[0].VolumeMounts[0].Name)
	envs := map[string]corev1.EnvVar{}
	for _, e := range p.Containers[0].Env {
		envs[e.Name] = e
	}
	assert.Contains(t, envs, "ES_TLS")
	assert.Contains(t, envs, "ES_TLS_CA")
	assert.Contains(t, envs, "ES_TLS_KEY")
	assert.Contains(t, envs, "ES_TLS_CERT")
}
