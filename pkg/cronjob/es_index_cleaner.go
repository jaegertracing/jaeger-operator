package cronjob

import (
	"fmt"
	"strconv"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func CreateEsIndexCleaner(jaeger *v1alpha1.Jaeger) *batchv1beta1.CronJob {
	esUrls := getEsHostname(jaeger.Spec.Storage.Options.Map())
	trueVar := true
	name := fmt.Sprintf("%s-es-index-cleaner", jaeger.Name)
	return &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: jaeger.Spec.Storage.EsIndexCleaner.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Image: jaeger.Spec.Storage.EsIndexCleaner.Image,
									Name:  name,
									Env:   []v1.EnvVar{{Name: "INDEX_PREFIX", Value: jaeger.Spec.Storage.Options.Map()["es.index-prefix"]}},
									Args:  []string{strconv.Itoa(jaeger.Spec.Storage.EsIndexCleaner.NumberOfDays), esUrls},
								},
							},
							RestartPolicy: v1.RestartPolicyNever,
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"prometheus.io/scrape":    "false",
								"sidecar.istio.io/inject": "false",
							},
						},
					},
				},
			},
		},
	}
}

// return first ES hostname from options map
func getEsHostname(opts map[string]string) string {
	urls, ok := opts["es.server-urls"]
	if !ok {
		return ""
	}
	urlArr := strings.Split(urls, ",")
	if len(urlArr) == 0 {
		return ""
	}
	return urlArr[0]
}
