package inventory

import (
	batchv1beta1 "k8s.io/api/batch/v1beta1"
)

// CronJob represents the inventory of cronjobs based on the current and desired states
type CronJob struct {
	Create []batchv1beta1.CronJob
	Update []batchv1beta1.CronJob
	Delete []batchv1beta1.CronJob
}

// ForCronJobs builds an inventory of cronjobs based on the existing and desired states
func ForCronJobs(existing []batchv1beta1.CronJob, desired []batchv1beta1.CronJob) CronJob {
	update := []batchv1beta1.CronJob{}
	mcreate := jobsMap(desired)
	mdelete := jobsMap(existing)

	for k, v := range mcreate {
		if _, ok := mdelete[k]; ok {
			update = append(update, v)
			delete(mcreate, k)
			delete(mdelete, k)
		}
	}

	return CronJob{
		Create: jobsList(mcreate),
		Update: update,
		Delete: jobsList(mdelete),
	}
}

func jobsMap(deps []batchv1beta1.CronJob) map[string]batchv1beta1.CronJob {
	m := map[string]batchv1beta1.CronJob{}
	for _, d := range deps {
		m[d.Name] = d
	}
	return m
}

func jobsList(m map[string]batchv1beta1.CronJob) []batchv1beta1.CronJob {
	l := []batchv1beta1.CronJob{}
	for _, v := range m {
		l = append(l, v)
	}
	return l
}
