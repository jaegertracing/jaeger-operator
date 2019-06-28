package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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

func TestCreateStreamingDeployment(t *testing.T) {
	name := "TestCreateStreamingDeployment"
	c := newStreamingStrategy(v1.NewJaeger(types.NamespacedName{Name: name}))
	assertDeploymentsAndServicesForStreaming(t, name, c, false, false, false)
}

func TestCreateStreamingDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateStreamingDeploymentOnOpenShift"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	normalize(jaeger)

	c := newStreamingStrategy(jaeger)
	assertDeploymentsAndServicesForStreaming(t, name, c, false, true, false)
}

func TestCreateStreamingDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateStreamingDeploymentWithDaemonSetAgent"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newStreamingStrategy(j)
	assertDeploymentsAndServicesForStreaming(t, name, c, true, false, false)
}

func TestCreateStreamingDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateStreamingDeploymentWithUIConfigMap"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.UI.Options = v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newStreamingStrategy(j)
	assertDeploymentsAndServicesForStreaming(t, name, c, false, false, true)
}

func TestStreamingOptionsArePassed(t *testing.T) {
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
			Strategy: "streaming",
			Collector: v1.JaegerCollectorSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.producer.topic": "mytopic",
				}),
			},
			Ingester: v1.JaegerIngesterSpec{
				Options: v1.NewOptions(map[string]interface{}{
					"kafka.consumer.topic":    "mytopic",
					"kafka.consumer.group-id": "mygroup",
				}),
			},
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
		// Only the query and ingester should have the ES parameters defined
		var escount int
		for _, arg := range args {
			if strings.Contains(arg, "es.") {
				escount++
			}
		}
		if strings.Contains(dep.Name, "collector") {
			// Including parameters for sampling config and kafka topic
			assert.Len(t, args, 2)
			assert.Equal(t, 0, escount)
		} else if strings.Contains(dep.Name, "ingester") {
			// Including parameters for ES and kafka topic
			assert.Len(t, args, 5)
			assert.Equal(t, 3, escount)

		} else {
			// Including parameters for ES only
			assert.Len(t, args, 3)
			assert.Equal(t, 3, escount)
		}
	}
}

func TestDelegateStreamingDependencies(t *testing.T) {
	// for now, we just have storage dependencies
	j := v1.NewJaeger(types.NamespacedName{Name: "TestDelegateStreamingDependencies"})
	c := newStreamingStrategy(j)
	assert.Equal(t, c.Dependencies(), storage.Dependencies(j))
}

func assertDeploymentsAndServicesForStreaming(t *testing.T, name string, s S, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
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

	serviceAccounts := map[string]bool{}
	if hasOAuthProxy {
		serviceAccounts[fmt.Sprintf("%s-ui-proxy", name)] = false
	}

	configMaps := map[string]bool{}
	if hasConfigMap {
		configMaps[fmt.Sprintf("%s-ui-configuration", name)] = false
	}
	assertHasAllObjects(t, name, s, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps)
}

func TestSparkDependenciesStreaming(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1.Jaeger) S {
		return newStreamingStrategy(jaeger)
	})
}

func TestEsIndexClenarStreaming(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1.Jaeger) S {
		return newStreamingStrategy(jaeger)
	})
}

func TestAgentSidecarIsInjectedIntoQueryForStreaming(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "TestAgentSidecarIsInjectedIntoQueryForStreaming"})
	c := newStreamingStrategy(j)
	for _, dep := range c.Deployments() {
		if strings.HasSuffix(dep.Name, "-query") {
			assert.Equal(t, 2, len(dep.Spec.Template.Spec.Containers))
			assert.Equal(t, "jaeger-agent", dep.Spec.Template.Spec.Containers[1].Name)
		}
	}
}
