package tls

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// TLSConfig represents a TLS configmap
type TLSConfig struct {
	jaeger *v1.Jaeger
}

// NewTLSConfig builds a new TLSConfig struct based on the given spec
func NewTLSConfig(jaeger *v1.Jaeger) *TLSConfig {
	return &TLSConfig{jaeger: jaeger}
}

// Get returns a configmap specification for the current instance
// TODO(@annanay25): Make this a global util function with default tags
func (t *TLSConfig) Get() *corev1.ConfigMap {
	t.jaeger.Logger().Debug("Assembling the TLS configmap")
	trueVar := true

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-collector-tls-config", t.jaeger.Name),
			Namespace: t.jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       fmt.Sprintf("%s-tls-configuration", t.jaeger.Name),
				"app.kubernetes.io/instance":   t.jaeger.Name,
				"app.kubernetes.io/component":  "tls-configuration",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
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
	cm.Annotations["service.beta.openshift.io/inject-cabundle"] = "true"
	return cm
}

// Update will modify the supplied common spec and options to include
// support for the TLS configmap if appropriate
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec, options *[]string) {
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
