package tls

import (
	"fmt"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Config represents a TLS configmap
type Config struct {
	jaeger *v1.Jaeger
}

// NewConfig builds a new Config struct based on the given spec
func NewConfig(jaeger *v1.Jaeger) *Config {
	return &Config{jaeger: jaeger}
}

// Get returns a configmap specification for the current instance
func (t *Config) Get() *corev1.ConfigMap {
	t.jaeger.Logger().Debug("Assembling the TLS configmap")
	trueVar := true

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-collector-tls-config", t.jaeger.Name),
			Namespace:   t.jaeger.Namespace,
			Annotations: map[string]string{"service.beta.openshift.io/inject-cabundle": "true"},
			Labels:      util.Labels(fmt.Sprintf("%s-tls-configuration", t.jaeger.Name), "tls-configuration", *t.jaeger),
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: t.jaeger.APIVersion,
					Kind:       t.jaeger.Kind,
					Name:       t.jaeger.Name,
					UID:        t.jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
	}
}

// Update will modify the supplied common spec and options to include
// support for the TLS configmap if appropriate
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec, options *[]string) {
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return
	}

	volume := corev1.Volume{
		Name: configurationVolumeName(jaeger),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-tls-configuration", jaeger.Name),
				},
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      configurationVolumeName(jaeger),
		MountPath: "/etc/config",
		ReadOnly:  true,
	}
	commonSpec.Volumes = append(commonSpec.Volumes, volume)
	commonSpec.VolumeMounts = append(commonSpec.VolumeMounts, volumeMount)
	*options = append(*options, "--collector.grpc.tls=true")
	*options = append(*options, "--collector.grpc.tls.cert=/etc/config/service-ca.crt")
}

func configurationVolumeName(jaeger *v1.Jaeger) string {
	return util.DNSName(fmt.Sprintf("%s-tls-configuration-volume", jaeger.Name))
}
