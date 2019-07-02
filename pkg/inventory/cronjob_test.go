package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCronJobInventory(t *testing.T) {
	toCreate := batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: "0 1 * * *",
		},
	}
	updated := batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: "0 2 * * *",
		},
	}
	toDelete := batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []batchv1beta1.CronJob{toUpdate, toDelete}
	desired := []batchv1beta1.CronJob{updated, toCreate}

	inv := ForCronJobs(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].Name)
	assert.Equal(t, "0 2 * * *", inv.Update[0].Spec.Schedule)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].Name)
}

func TestCronJobInventoryWithSameNameInstances(t *testing.T) {
	create := []batchv1beta1.CronJob{{
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

	inv := ForCronJobs([]batchv1beta1.CronJob{}, create)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, create[0])
	assert.Contains(t, inv.Create, create[1])
	assert.Len(t, inv.Update, 0)
	assert.Len(t, inv.Delete, 0)
}
