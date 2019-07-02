package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressInventory(t *testing.T) {
	toCreate := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: "service-a",
			},
		},
	}
	updated := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: "service-b",
			},
		},
	}
	toDelete := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []v1beta1.Ingress{toUpdate, toDelete}
	desired := []v1beta1.Ingress{updated, toCreate}

	inv := ForIngresses(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "service-b", inv.Update[0].Spec.Backend.ServiceName)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestIngressInventoryWithSameNameInstances(t *testing.T) {
	create := []v1beta1.Ingress{{
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

	inv := ForIngresses([]v1beta1.Ingress{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
