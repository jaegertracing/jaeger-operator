package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestHorizontalPodAutoscalerInventory(t *testing.T) {
	toCreate := autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}
	toUpdate := autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-update",
			Namespace: "tenant1",
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 1,
		},
	}
	updated := autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Namespace:   "tenant1",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 2,
		},
	}
	toDelete := autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-delete",
			Namespace: "tenant1",
		},
	}

	existing := []autoscalingv2beta2.HorizontalPodAutoscaler{toUpdate, toDelete}
	desired := []autoscalingv2beta2.HorizontalPodAutoscaler{updated, toCreate}

	inv := ForHorizontalPodAutoscalers(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, int32(2), inv.Update[0].Spec.MaxReplicas)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestHorizontalPodAutoscalerInventoryWithSameNameInstances(t *testing.T) {
	create := []autoscalingv2beta2.HorizontalPodAutoscaler{{
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

	inv := ForHorizontalPodAutoscalers([]autoscalingv2beta2.HorizontalPodAutoscaler{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, create, create[0])
	assert.Contains(t, create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}

func TestHorizontalPodAutoscalerInventoryNewWithSameNameAsExisting(t *testing.T) {
	create := autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}

	existing := []autoscalingv2beta2.HorizontalPodAutoscaler{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant2",
		},
	}}

	util.InitObjectMeta(&existing[0])
	inv := ForHorizontalPodAutoscalers(existing, append(existing, create))

	assert.Len(t, inv.Create, 1)
	assert.Equal(t, inv.Create[0], create)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, inv.Update[0], existing[0])

	assert.Len(t, inv.Delete, 0)
}
