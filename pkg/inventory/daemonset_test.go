package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDaemonSetInventory(t *testing.T) {
	toCreate := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: appsv1.DaemonSetSpec{
			MinReadySeconds: 1,
		},
	}
	updated := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: appsv1.DaemonSetSpec{
			MinReadySeconds: 2,
		},
	}
	toDelete := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []appsv1.DaemonSet{toUpdate, toDelete}
	desired := []appsv1.DaemonSet{updated, toCreate}

	inv := ForDaemonSets(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, int32(2), inv.Update[0].Spec.MinReadySeconds)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestDaemonSetInventoryWithSameNameInstances(t *testing.T) {
	create := []appsv1.DaemonSet{{
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

	inv := ForDaemonSets([]appsv1.DaemonSet{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
