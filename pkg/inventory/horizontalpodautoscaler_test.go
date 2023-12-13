package inventory

import (
	"testing"

	kruntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestHorizontalPodAutoscalerInventory(t *testing.T) {
	toCreate := autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}
	toUpdate := autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-update",
			Namespace: "tenant1",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 1,
		},
	}
	updated := autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Namespace:   "tenant1",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MaxReplicas: 2,
		},
	}
	toDelete := autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-delete",
			Namespace: "tenant1",
		},
	}

	existing := []kruntime.Object{&toUpdate, &toDelete}
	desired := []kruntime.Object{&updated, &toCreate}

	inv := ForHorizontalPodAutoscalers(existing, desired)
	assert.Len(t, inv.Create, 1)
	create := inv.Create[0].(*autoscalingv2.HorizontalPodAutoscaler)
	assert.Equal(t, "to-create", create.Name)

	assert.Len(t, inv.Update, 1)
	update := inv.Update[0].(*autoscalingv2.HorizontalPodAutoscaler)
	assert.Equal(t, "to-update", update.Name)
	assert.Equal(t, int32(2), update.Spec.MaxReplicas)

	assert.Len(t, inv.Delete, 1)
	delete := inv.Delete[0].(*autoscalingv2.HorizontalPodAutoscaler)
	assert.Equal(t, "to-delete", delete.Name)
}

func TestHorizontalPodAutoscalerInventoryWithSameNameInstances(t *testing.T) {
	create := []kruntime.Object{
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "to-create",
				Namespace: "tenant1",
			},
		},
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "to-create",
				Namespace: "tenant2",
			},
		},
	}

	inv := ForHorizontalPodAutoscalers([]kruntime.Object{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, create, create[0])
	assert.Contains(t, create, create[1])
	assert.Empty(t, inv.Update)
	assert.Empty(t, inv.Delete)
}

func TestHorizontalPodAutoscalerInventoryNewWithSameNameAsExisting(t *testing.T) {
	viper.Set(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2)
	create := []kruntime.Object{&autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}}

	existing := []kruntime.Object{&autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant2",
		},
	}}

	existingObject := existing[0].(*autoscalingv2.HorizontalPodAutoscaler)

	util.InitObjectMeta(existingObject)
	inv := ForHorizontalPodAutoscalers(existing, append(existing, create...))

	assert.Len(t, inv.Create, 1)
	assert.Equal(t, inv.Create[0], create[0].(*autoscalingv2.HorizontalPodAutoscaler))

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, inv.Update[0], existing[0])

	assert.Empty(t, inv.Delete)
}

func TestHorizontalPodAutoscalerInventoryNewWithSameNameAsExistingBeta2(t *testing.T) {
	viper.Set(v1.FlagAutoscalingVersion, v1.FlagAutoscalingVersionV2Beta2)
	create := []kruntime.Object{&autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant1",
		},
	}}

	existing := []kruntime.Object{&autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-create",
			Namespace: "tenant2",
		},
	}}

	existingObject := existing[0].(*autoscalingv2beta2.HorizontalPodAutoscaler)

	util.InitObjectMeta(existingObject)
	inv := ForHorizontalPodAutoscalers(existing, append(existing, create...))

	assert.Len(t, inv.Create, 1)
	assert.Equal(t, inv.Create[0], create[0].(*autoscalingv2beta2.HorizontalPodAutoscaler))

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, inv.Update[0], existing[0])

	assert.Empty(t, inv.Delete)
}
