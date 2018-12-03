package sampling

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

const (
	defaultSamplingStrategy = "{\"default_strategy\":{\"param\":1,\"type\":\"probabilistic\"}}"
)

// Config represents a sampling configmap
type Config struct {
	jaeger *v1alpha1.Jaeger
}

// NewConfig builds a new Config struct based on the given spec
func NewConfig(jaeger *v1alpha1.Jaeger) *Config {
	return &Config{jaeger: jaeger}
}

// Get returns a configmap specification for the current instance
func (u *Config) Get() *v1.ConfigMap {
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

	logrus.WithField("instance", u.jaeger.Name).Debug("Assembling the Sampling configmap")
	trueVar := true

	data := map[string]string{
		"sampling": string(jsonObject),
	}

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sampling-configuration", u.jaeger.Name),
			Namespace: u.jaeger.Namespace,
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
func Update(jaeger *v1alpha1.Jaeger, commonSpec *v1alpha1.JaegerCommonSpec, options *[]string) {

	volume := v1.Volume{
		Name: fmt.Sprintf("%s-sampling-configuration-volume", jaeger.Name),
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: fmt.Sprintf("%s-sampling-configuration", jaeger.Name),
				},
				Items: []v1.KeyToPath{
					v1.KeyToPath{
						Key:  "sampling",
						Path: "sampling.json",
					},
				},
			},
		},
	}
	volumeMount := v1.VolumeMount{
		Name:      fmt.Sprintf("%s-sampling-configuration-volume", jaeger.Name),
		MountPath: "/etc/jaeger/sampling",
		ReadOnly:  true,
	}
	commonSpec.Volumes = append(commonSpec.Volumes, volume)
	commonSpec.VolumeMounts = append(commonSpec.VolumeMounts, volumeMount)
	*options = append(*options, "--sampling.strategies-file=/etc/jaeger/sampling/sampling.json")
}
