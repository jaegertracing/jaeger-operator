package inventory

import (
	"testing"

	osv1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRouteInventory(t *testing.T) {
	toCreate := osv1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := osv1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: osv1.RouteSpec{
			Host: "v1.example.com",
		},
	}
	updated := osv1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: osv1.RouteSpec{
			Host: "v2.example.com",
		},
	}
	toDelete := osv1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []osv1.Route{toUpdate, toDelete}
	desired := []osv1.Route{updated, toCreate}

	inv := ForRoutes(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "v2.example.com", inv.Update[0].Spec.Host)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestRouteInventoryWithSameNameInstances(t *testing.T) {
	create := []osv1.Route{{
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

	inv := ForRoutes([]osv1.Route{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
