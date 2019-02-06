package strategy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

func TestWithDeployments(t *testing.T) {
	c := New().WithDeployments([]appsv1.Deployment{{}})
	assert.Len(t, c.Deployments(), 1)
}

func TestWithDependencies(t *testing.T) {
	c := New().WithDependencies([]batchv1.Job{{}})
	assert.Len(t, c.Dependencies(), 1)
}
