package inventory

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// Always test with v1.  It is available at compile time and is exactly the same as v1beta1
	viper.SetDefault(v1.FlagCronJobsVersion, v1.FlagCronJobsVersionBatchV1)
}

func TestCronJobInventory(t *testing.T) {
	toCreate := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-create",
		},
	}
	toUpdate := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-update",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 1 * * *",
		},
	}
	updated := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-update",
			Annotations: map[string]string{"gopher": "jaeger"},
			Labels:      map[string]string{"gopher": "jaeger"},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 2 * * *",
		},
	}
	toDelete := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "to-delete",
		},
	}

	existing := []runtime.Object{toUpdate, toDelete}
	desired := []runtime.Object{updated, toCreate}

	inv := ForCronJobs(existing, desired)
	assert.Len(t, inv.Create, 1)
	assert.Equal(t, "to-create", inv.Create[0].(*batchv1.CronJob).Name)

	assert.Len(t, inv.Update, 1)
	assert.Equal(t, "to-update", inv.Update[0].(*batchv1.CronJob).Name)
	assert.Equal(t, "0 2 * * *", inv.Update[0].(*batchv1.CronJob).Spec.Schedule)

	assert.Len(t, inv.Delete, 1)
	assert.Equal(t, "to-delete", inv.Delete[0].(*batchv1.CronJob).Name)
}

func TestCronJobInventoryWithSameNameInstances(t *testing.T) {
	create := []batchv1.CronJob{{
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

	var o1 runtime.Object = &create[0]
	var o2 runtime.Object = &create[1]
	createObj := []runtime.Object{o1, o2}

	inv := ForCronJobs([]runtime.Object{}, createObj)
	assert.Len(t, inv.Create, 2)
	assert.Contains(t, inv.Create, createObj[0])
	assert.Contains(t, inv.Create, createObj[1])
	assert.Empty(t, inv.Update)
	assert.Empty(t, inv.Delete)
}

func TestCronJobInventoryWithRepeats(t *testing.T) {
	job1 := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-jaeger-spark-dependencies",
		},
	}
	job2 := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-jaeger-es-index-cleaner",
		},
	}

	existing := []runtime.Object{job1}
	desired := []runtime.Object{job1, job2}
	inventory := ForCronJobs(existing, desired)
	assert.Len(t, inventory.Create, 1)
	assert.Contains(t, inventory.Create, job2)
	assert.Len(t, inventory.Update, 1)

	assert.Equal(t, job1.Name, inventory.Update[0].(*batchv1.CronJob).Name)
	for _, v := range inventory.Update {
		fmt.Printf(v.(*batchv1.CronJob).Name)
	}
	assert.Empty(t, inventory.Delete)
}
