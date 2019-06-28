package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestDeploymentInventory(t *testing.T) {
	toCreate := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}
	toUpdate := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-update",
			Namespace: "tenant1",
		},
		Spec: appsv1.DeploymentSpec{
			MinReadySeconds: 1,
		},
	}
	updated := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Namespace:   "tenant1",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: appsv1.DeploymentSpec{
			MinReadySeconds: 2,
		},
	}
	toDelete := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-delete",
			Namespace: "tenant1",
		},
	}

	existing := []appsv1.Deployment{toUpdate, toDelete}
	desired := []appsv1.Deployment{updated, toCreate}

	inv := ForDeployments(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, int32(2), inv.Update[0].Spec.MinReadySeconds)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestDeploymentInventoryWithSameNameInstances(t *testing.T) {
	create := []appsv1.Deployment{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant2",
		},
	}}

	inv := ForDeployments([]appsv1.Deployment{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, create, create[0])
	assert.Contains(t, create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}

func TestDeploymentInventoryNewWithSameNameAsExisting(t *testing.T) {
	create := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}

	existing := []appsv1.Deployment{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant2",
		},
	}}

	util.InitObjectMeta(&existing[0])
	inv := ForDeployments(existing, append(existing, create))

	assert.Len(t, inv.Create, 1)
	assert.Equal(t, inv.Create[0], create)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, inv.Update[0], existing[0])

	assert.Len(t, inv.Delete, 0)
}
