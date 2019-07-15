package strategy

import (
	"testing"

	osv1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"

	esv1 "github.com/jaegertracing/jaeger-operator/pkg/storage/elasticsearch/v1"
)

func TestWithAccounts(t *testing.T) {
	c := New().WithAccounts([]v1.ServiceAccount{{}})
	assert.Len(t, c.Accounts(), 1)
}

func TestWithClusterRoleBindings(t *testing.T) {
	c := New().WithClusterRoleBindings([]rbac.ClusterRoleBinding{{}})
	assert.Len(t, c.ClusterRoleBindings(), 1)
}

func TestWithConfigMaps(t *testing.T) {
	c := New().WithConfigMaps([]v1.ConfigMap{{}})
	assert.Len(t, c.ConfigMaps(), 1)
}

func TestWithCronJobs(t *testing.T) {
	c := New().WithCronJobs([]batchv1beta1.CronJob{{}})
	assert.Len(t, c.CronJobs(), 1)
}

func TestWithDaemonSets(t *testing.T) {
	c := New().WithDaemonSets([]appsv1.DaemonSet{{}})
	assert.Len(t, c.DaemonSets(), 1)
}

func TestWithDependencies(t *testing.T) {
	c := New().WithDependencies([]batchv1.Job{{}})
	assert.Len(t, c.Dependencies(), 1)
}

func TestWithDeployments(t *testing.T) {
	c := New().WithDeployments([]appsv1.Deployment{{}})
	assert.Len(t, c.Deployments(), 1)
}

func TestWithElasticsearches(t *testing.T) {
	c := New().WithElasticsearches([]esv1.Elasticsearch{{}})
	assert.Len(t, c.Elasticsearches(), 1)
}

func TestWithIngresses(t *testing.T) {
	c := New().WithIngresses([]v1beta1.Ingress{{}})
	assert.Len(t, c.Ingresses(), 1)
}

func TestWithRoutes(t *testing.T) {
	c := New().WithRoutes([]osv1.Route{{}})
	assert.Len(t, c.Routes(), 1)
}

func TestWithServices(t *testing.T) {
	c := New().WithServices([]v1.Service{{}})
	assert.Len(t, c.Services(), 1)
}

func TestWithSecrets(t *testing.T) {
	c := New().WithSecrets([]v1.Secret{{}})
	assert.Len(t, c.Secrets(), 1)
}
