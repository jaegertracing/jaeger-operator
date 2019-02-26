package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
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
		},
	}
	updated := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: v1.ServiceSpec{
			ExternalName: "v2.example.com",
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
}
