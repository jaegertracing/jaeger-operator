package strategy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
)

func TestWithAccounts(t *testing.T) {
	c := New().WithAccounts([]v1.ServiceAccount{{}})
	assert.Len(t, c.Accounts(), 1)
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
