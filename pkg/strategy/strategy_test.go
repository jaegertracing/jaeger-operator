package strategy

import (
	"testing"

	osconsolev1 "github.com/openshift/api/console/v1"
	osv1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"

	kafkav1beta2 "github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

func TestWithAccounts(t *testing.T) {
	c := New().WithAccounts([]v1.ServiceAccount{{}})
	assert.Len(t, c.Accounts(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithClusterRoleBindings(t *testing.T) {
	c := New().WithClusterRoleBindings([]rbac.ClusterRoleBinding{{}})
	assert.Len(t, c.ClusterRoleBindings(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithConsoleLinks(t *testing.T) {
	c := New().WithConsoleLinks([]osconsolev1.ConsoleLink{{}})
	assert.Len(t, c.ConsoleLinks([]osv1.Route{{}}), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithConfigMaps(t *testing.T) {
	c := New().WithConfigMaps([]v1.ConfigMap{{}})
	assert.Len(t, c.ConfigMaps(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithCronJobs(t *testing.T) {
	c := New().WithCronJobs([]batchv1beta1.CronJob{{}})
	assert.Len(t, c.CronJobs(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithDaemonSets(t *testing.T) {
	c := New().WithDaemonSets([]appsv1.DaemonSet{{}})
	assert.Len(t, c.DaemonSets(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithDependencies(t *testing.T) {
	c := New().WithDependencies([]batchv1.Job{{}})
	assert.Len(t, c.Dependencies(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithDeployments(t *testing.T) {
	c := New().WithDeployments([]appsv1.Deployment{{}})
	assert.Len(t, c.Deployments(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithElasticsearches(t *testing.T) {
	c := New().WithElasticsearches([]esv1.Elasticsearch{{}})
	assert.Len(t, c.Elasticsearches(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithIngresses(t *testing.T) {
	c := New().WithIngresses([]networkingv1.Ingress{{}})
	assert.Len(t, c.Ingresses(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithKafkas(t *testing.T) {
	c := New().WithKafkas([]kafkav1beta2.Kafka{{}})
	assert.Len(t, c.Kafkas(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithKafkaUsers(t *testing.T) {
	c := New().WithKafkaUsers([]kafkav1beta2.KafkaUser{{}})
	assert.Len(t, c.KafkaUsers(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithRoutes(t *testing.T) {
	c := New().WithRoutes([]osv1.Route{{}})
	assert.Len(t, c.Routes(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithServices(t *testing.T) {
	c := New().WithServices([]v1.Service{{}})
	assert.Len(t, c.Services(), 1)
	assert.Len(t, c.All(), 1)
}

func TestWithSecrets(t *testing.T) {
	c := New().WithSecrets([]v1.Secret{{}})
	assert.Len(t, c.Secrets(), 1)
	assert.Len(t, c.All(), 1)
}
