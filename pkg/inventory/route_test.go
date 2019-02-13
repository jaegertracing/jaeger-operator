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
	}
	toDelete := osv1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []osv1.Route{toUpdate, toDelete}
	desired := []osv1.Route{toUpdate, toCreate}

	inv := ForRoutes(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
