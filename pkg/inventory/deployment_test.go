package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentInventory(t *testing.T) {
	depToCreate := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dep-to-create",
		},
	}
	depToUpdate := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dep-to-update",
		},
	}
	depToDelete := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dep-to-delete",
		},
	}

	existing := []appsv1.Deployment{depToUpdate, depToDelete}
	desired := []appsv1.Deployment{depToUpdate, depToCreate}

	depInventory := ForDeployments(existing, desired)
	assert.Equal(t, "dep-to-create", depInventory.Create[0].Name)
	assert.Len(t, depInventory.Create, 1)

	assert.Equal(t, "dep-to-update", depInventory.Update[0].Name)
	assert.Len(t, depInventory.Update, 1)

	assert.Equal(t, "dep-to-delete", depInventory.Delete[0].Name)
	assert.Len(t, depInventory.Delete, 1)
}
