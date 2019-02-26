package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapInventory(t *testing.T) {
	toCreate := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Data: map[string]string{
			"field": "foo",
		},
	}
	updated := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Data: map[string]string{
			"field": "bar",
		},
	}
	toDelete := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []v1.ConfigMap{toUpdate, toDelete}
	desired := []v1.ConfigMap{updated, toCreate}

	inv := ForConfigMaps(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "bar", inv.Update[0].Data["field"])

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}
