package inventory

import (
	"fmt"

	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"

	batchv1beta1 "k8s.io/api/batch/v1beta1"

	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// CronJob represents the inventory of cronjobs based on the current and desired states
type CronJob struct {
	Create []runtime.Object
	Update []runtime.Object
	Delete []runtime.Object
}

// ForCronJobs builds an inventory of cronjobs based on the existing and desired states
func ForCronJobs(existing []runtime.Object, desired []runtime.Object) CronJob {
	update := []runtime.Object{}
	desiredCronjobsMap := jobsMap(desired)
	existingCronJobsMap := jobsMap(existing)

	cronjobsVersion := viper.GetString(v1.FlagCronJobsVersion)

	for desiredKey, desiredValue := range desiredCronjobsMap {
		if existingValue, ok := existingCronJobsMap[desiredKey]; ok {
			if cronjobsVersion == v1.FlagCronJobsVersionBatchV1Beta1 {
				t1 := existingValue.(*batchv1beta1.CronJob)
				v1 := desiredValue.(*batchv1beta1.CronJob)
				tp := t1.DeepCopy()
				util.InitObjectMeta(tp)

				// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
				tp.Spec = v1.Spec
				tp.ObjectMeta.OwnerReferences = v1.ObjectMeta.OwnerReferences

				for k, v := range v1.ObjectMeta.Annotations {
					tp.ObjectMeta.Annotations[k] = v
				}

				for k, v := range v1.ObjectMeta.Labels {
					tp.ObjectMeta.Labels[k] = v
				}

				update = append(update, tp)
			} else {
				t1 := existingValue.(*batchv1.CronJob)
				v1 := desiredValue.(*batchv1.CronJob)
				tp := t1.DeepCopy()
				util.InitObjectMeta(tp)

				// we can't blindly DeepCopyInto, so, we select what we bring from the new to the old object
				tp.Spec = v1.Spec
				tp.ObjectMeta.OwnerReferences = v1.ObjectMeta.OwnerReferences

				for k, v := range v1.ObjectMeta.Annotations {
					tp.ObjectMeta.Annotations[k] = v
				}

				for k, v := range v1.ObjectMeta.Labels {
					tp.ObjectMeta.Labels[k] = v
				}

				update = append(update, tp)
			}

			delete(desiredCronjobsMap, desiredKey)
			delete(existingCronJobsMap, desiredKey)
		}
	}

	result := CronJob{
		Create: jobsList(desiredCronjobsMap),
		Update: update,
		Delete: jobsList(existingCronJobsMap),
	}
	return result
}

func jobsMap(deps []runtime.Object) map[string]runtime.Object {
	m := map[string]runtime.Object{}
	var key string
	cronjobsVersion := viper.GetString(v1.FlagCronJobsVersion)

	for _, d := range deps {
		if cronjobsVersion == v1.FlagCronJobsVersionBatchV1Beta1 {
			cj := d.(*batchv1beta1.CronJob)
			key = fmt.Sprintf("%s.%s", cj.Namespace, cj.Name)
		} else {
			cj := d.(*batchv1.CronJob)
			key = fmt.Sprintf("%s.%s", cj.Namespace, cj.Name)
		}
		m[key] = d
	}
	return m
}

func jobsList(m map[string]runtime.Object) []runtime.Object {
	l := []runtime.Object{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
