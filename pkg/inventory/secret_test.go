package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretInventory(t *testing.T) {
	toCreate := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		StringData: map[string]string{
			"field": "foo",
		},
	}
	updated := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		StringData: map[string]string{
			"field": "bar",
		},
	}
	toDelete := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []v1.Secret{toUpdate, toDelete}
	desired := []v1.Secret{updated, toCreate}

	inv := ForSecrets(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "bar", inv.Update[0].StringData["field"])

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
