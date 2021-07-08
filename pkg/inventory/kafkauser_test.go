package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/kafka/v1beta2"
)

func TestKafkaUserInventory(t *testing.T) {
	toCreate := v1beta2.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := v1beta2.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: v1beta2.KafkaUserSpec{
			v1.NewFreeForm(map[string]interface{}{
				"key": "original",
			}),
		},
	}
	updated := v1beta2.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: v1beta2.KafkaUserSpec{
			v1.NewFreeForm(map[string]interface{}{
				"key": "changed",
			}),
		},
	}
	toDelete := v1beta2.KafkaUser{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []v1beta2.KafkaUser{toUpdate, toDelete}
	desired := []v1beta2.KafkaUser{updated, toCreate}

	inv := ForKafkaUsers(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	contentMap, err := inv.Update[0].Spec.GetMap()
	assert.NoError(t, err)
	assert.Equal(t, "changed", contentMap["key"])

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestKafkaUserInventoryWithSameNameInstances(t *testing.T) {
	create := []v1beta2.KafkaUser{{
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

	inv := ForKafkaUsers([]v1beta2.KafkaUser{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
