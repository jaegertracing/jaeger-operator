package sampling

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	defaultSamplingStrategy = "{\"default_strategy\":{\"param\":1,\"type\":\"probabilistic\"}}"
)

// Config represents a sampling configmap
type Config struct {
	jaeger *v1.Jaeger
}

// NewConfig builds a new Config struct based on the given spec
func NewConfig(jaeger *v1.Jaeger) *Config {
	return &Config{jaeger: jaeger}
}

// Get returns a configmap specification for the current instance
func (u *Config) Get() *corev1.ConfigMap {
	var jsonObject []byte
	var err error

	// Check for empty map
	if u.jaeger.Spec.Sampling.Options.IsEmpty() {
		jsonObject = []byte(defaultSamplingStrategy)
	} else {
		jsonObject, err = u.jaeger.Spec.Sampling.Options.MarshalJSON()
	}

	if err != nil {
		return nil
	}

	u.jaeger.Logger().Debug("Assembling the Sampling configmap")
	trueVar := true

	data := map[string]string{
		"sampling": string(jsonObject),
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sampling-configuration", u.jaeger.Name),
			Namespace: u.jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       fmt.Sprintf("%s-sampling-configuration", u.jaeger.Name),
				"app.kubernetes.io/instance":   u.jaeger.Name,
				"app.kubernetes.io/component":  "sampling-configuration",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: u.jaeger.APIVersion,
					Kind:       u.jaeger.Kind,
					Name:       u.jaeger.Name,
					UID:        u.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Data: data,
	}
}

// Update will modify the supplied common spec and options to include
// support for the Sampling configmap.
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec, options *[]string) {

	volume := corev1.Volume{
		Name: samplingConfigVolumeName(jaeger),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-sampling-configuration", jaeger.Name),
				},
				Items: []corev1.KeyToPath{
					corev1.KeyToPath{
						Key:  "sampling",
						Path: "sampling.json",
					},
				},
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      samplingConfigVolumeName(jaeger),
		MountPath: "/etc/jaeger/sampling",
		ReadOnly:  true,
	}
	commonSpec.Volumes = append(commonSpec.Volumes, volume)
	commonSpec.VolumeMounts = append(commonSpec.VolumeMounts, volumeMount)
	*options = append(*options, "--sampling.strategies-file=/etc/jaeger/sampling/sampling.json")
}

func samplingConfigVolumeName(jaeger *v1.Jaeger) string {
	return util.DNSName(fmt.Sprintf("%s-sampling-configuration-volume", jaeger.Name))
}
