package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccountInventory(t *testing.T) {
	trueVar, falseVar := true, false

	toCreate := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		AutomountServiceAccountToken: &trueVar,
	}
	updated := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		AutomountServiceAccountToken: &falseVar,
	}
	toDelete := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []v1.ServiceAccount{toUpdate, toDelete}
	desired := []v1.ServiceAccount{updated, toCreate}

	inv := ForAccounts(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, &falseVar, inv.Update[0].AutomountServiceAccountToken)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
