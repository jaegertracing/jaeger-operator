package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
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
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		AutomountServiceAccountToken: &falseVar,
		Secrets:                      []v1.ObjectReference{{Kind: "Secret"}},
		ImagePullSecrets:             []v1.LocalObjectReference{{Name: "ImagePullSecret"}},
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

	// we do *not* set this in any of our current service accounts,
	// but this might be set by the cluster -- in this case,
	// we keep whatever is there, not touching the fields at all
	assert.Equal(t, &trueVar, inv.Update[0].AutomountServiceAccountToken)
	assert.Len(t, inv.Update[0].Secrets, 0)
	assert.Len(t, inv.Update[0].ImagePullSecrets, 0)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestAccountInventoryWithSameNameInstances(t *testing.T) {
	create := []v1.ServiceAccount{{
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

	inv := ForAccounts([]v1.ServiceAccount{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
