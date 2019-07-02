package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceInventory(t *testing.T) {
	toCreate := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: v1.ServiceSpec{
			ExternalName: "v1.example.com",
			ClusterIP:    "10.97.132.43", // got assigned by Kubernetes
		},
	}
	updated := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: v1.ServiceSpec{
			ExternalName: "v2.example.com",
			ClusterIP:    "", // will get assigned by Kubernetes
		},
	}
	toDelete := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []v1.Service{toUpdate, toDelete}
	desired := []v1.Service{updated, toCreate}

	inv := ForServices(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "v2.example.com", inv.Update[0].Spec.ExternalName)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)

	assert.Equal(t, toUpdate.Spec.ClusterIP, inv.Update[0].Spec.ClusterIP)
}

func TestServiceInventoryWithSameNameInstances(t *testing.T) {
	create := []v1.Service{{
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

	inv := ForServices([]v1.Service{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
