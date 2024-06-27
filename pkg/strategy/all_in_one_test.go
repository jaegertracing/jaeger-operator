package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/consolelink"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jaegertracing/jaeger-operator/pkg/storage"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func init() {
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}

func TestCreateAllInOneDeployment(t *testing.T) {
	name := "TestCreateAllInOneDeployment"
	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	c := newAllInOneStrategy(context.Background(), jaeger)
	assertDeploymentsAndServicesForAllInOne(t, jaeger, c, false, false, false)
}

func TestCreateAllInOneDeploymentOnOpenShift(t *testing.T) {
	viper.Set("platform", "openshift")
	defer viper.Reset()
	name := "TestCreateAllInOneDeploymentOnOpenShift"

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	normalize(context.Background(), jaeger)

	c := newAllInOneStrategy(context.Background(), jaeger)
	assertDeploymentsAndServicesForAllInOne(t, jaeger, c, false, true, false)
}

func TestCreateAllInOneDeploymentWithDaemonSetAgent(t *testing.T) {
	name := "TestCreateAllInOneDeploymentWithDaemonSetAgent"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.Agent.Strategy = "DaemonSet"

	c := newAllInOneStrategy(context.Background(), j)
	assertDeploymentsAndServicesForAllInOne(t, j, c, true, false, false)
}

func TestCreateAllInOneDeploymentWithUIConfigMap(t *testing.T) {
	name := "TestCreateAllInOneDeploymentWithUIConfigMap"

	j := v1.NewJaeger(types.NamespacedName{Name: name})
	j.Spec.UI.Options = v1.NewFreeForm(map[string]interface{}{
		"tracking": map[string]interface{}{
			"gaID": "UA-000000-2",
		},
	})

	c := newAllInOneStrategy(context.Background(), j)
	assertDeploymentsAndServicesForAllInOne(t, j, c, false, false, true)
}

func TestDelegateAllInOneDependencies(t *testing.T) {
	// for now, we just have storage dependencies
	j := v1.NewJaeger(types.NamespacedName{Name: "TestDelegateAllInOneDependencies"})
	c := newAllInOneStrategy(context.Background(), j)
	assert.Equal(t, c.Dependencies(), storage.Dependencies(j))
}

func TestNoAutoscaleForAllInOne(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	c := newAllInOneStrategy(context.Background(), j)
	assert.Empty(t, c.HorizontalPodAutoscalers())
}

func assertDeploymentsAndServicesForAllInOne(t *testing.T, instance *v1.Jaeger, s S, hasDaemonSet bool, hasOAuthProxy bool, hasConfigMap bool) {
	// TODO(jpkroehling): this func deserves a refactoring already
	name := instance.Name

	// we should have one deployment, named after the Jaeger's name (ObjectMeta.Name)
	deployments := map[string]bool{
		name: false,
	}

	daemonsets := map[string]bool{
		fmt.Sprintf("%s-agent-daemonset", name): !hasDaemonSet,
	}

	// and these services
	services := map[string]bool{
		fmt.Sprintf("%s-agent", strings.ToLower(name)):     false,
		fmt.Sprintf("%s-collector", strings.ToLower(name)): false,
		fmt.Sprintf("%s-query", strings.ToLower(name)):     false,
	}

	// the ingress rule, if we are not on openshift
	ingresses := map[string]bool{}
	routes := map[string]bool{}
	consoleLinks := map[string]bool{}
	if autodetect.OperatorConfiguration.GetPlatform() == autodetect.OpenShiftPlatform {
		routes[util.DNSName(name)] = false
		consoleLinks[consolelink.Name(instance)] = false

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
	assertHasAllObjects(t, name, s, deployments, daemonsets, services, ingresses, routes, serviceAccounts, configMaps, consoleLinks)
}

func TestSparkDependenciesAllInOne(t *testing.T) {
	testSparkDependencies(t, func(jaeger *v1.Jaeger) S {
		return newAllInOneStrategy(context.Background(), jaeger)
	})
}

func testSparkDependencies(t *testing.T, fce func(jaeger *v1.Jaeger) S) {
	trueVar := true
	tests := []struct {
		jaeger              *v1.Jaeger
		sparkCronJobEnabled bool
	}{
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type:         v1.JaegerESStorage,
				Dependencies: v1.JaegerDependenciesSpec{Enabled: &trueVar},
			},
		}}, sparkCronJobEnabled: true},
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type:         v1.JaegerCassandraStorage,
				Dependencies: v1.JaegerDependenciesSpec{Enabled: &trueVar},
			},
		}}, sparkCronJobEnabled: true},
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type:         v1.JaegerKafkaStorage,
				Dependencies: v1.JaegerDependenciesSpec{Enabled: &trueVar},
			},
		}}, sparkCronJobEnabled: false},
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage},
		}}, sparkCronJobEnabled: false},
	}
	for _, test := range tests {
		s := fce(test.jaeger)
		cronJobs := s.CronJobs()
		if test.sparkCronJobEnabled {
			assert.Len(t, cronJobs, 1)
		} else {
			assert.Empty(t, cronJobs)
		}
	}
}

func TestEsIndexCleanerAllInOne(t *testing.T) {
	testEsIndexCleaner(t, func(jaeger *v1.Jaeger) S {
		return newAllInOneStrategy(context.Background(), jaeger)
	})
}

func testEsIndexCleaner(t *testing.T, fce func(jaeger *v1.Jaeger) S) {
	trueVar := true
	days := 0
	tests := []struct {
		jaeger              *v1.Jaeger
		sparkCronJobEnabled bool
	}{
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type:           v1.JaegerESStorage,
				EsIndexCleaner: v1.JaegerEsIndexCleanerSpec{Enabled: &trueVar, NumberOfDays: &days},
			},
		}}, sparkCronJobEnabled: true},
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type:           v1.JaegerCassandraStorage,
				EsIndexCleaner: v1.JaegerEsIndexCleanerSpec{Enabled: &trueVar, NumberOfDays: &days},
			},
		}}, sparkCronJobEnabled: false},
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type:           v1.JaegerKafkaStorage,
				EsIndexCleaner: v1.JaegerEsIndexCleanerSpec{Enabled: &trueVar, NumberOfDays: &days},
			},
		}}, sparkCronJobEnabled: false},
		{jaeger: &v1.Jaeger{Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{Type: v1.JaegerESStorage},
		}}, sparkCronJobEnabled: false},
	}
	for _, test := range tests {
		s := fce(test.jaeger)
		cronJobs := s.CronJobs()
		if test.sparkCronJobEnabled {
			assert.Len(t, cronJobs, 1)
		} else {
			assert.Empty(t, cronJobs)
		}
	}
}
