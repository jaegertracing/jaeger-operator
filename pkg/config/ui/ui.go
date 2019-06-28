package configmap

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// UIConfig represents a UI configmap
type UIConfig struct {
	jaeger *v1.Jaeger
}

// NewUIConfig builds a new UIConfig struct based on the given spec
func NewUIConfig(jaeger *v1.Jaeger) *UIConfig {
	return &UIConfig{jaeger: jaeger}
}

// Get returns a configmap specification for the current instance
func (u *UIConfig) Get() *corev1.ConfigMap {
	// Check for empty map
	if u.jaeger.Spec.UI.Options.IsEmpty() {
		return nil
	}

	json, err := u.jaeger.Spec.UI.Options.MarshalJSON()
	if err != nil {
		return nil
	}

	u.jaeger.Logger().Debug("Assembling the UI configmap")
	trueVar := true
	data := map[string]string{
		"ui": string(json),
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ui-configuration", u.jaeger.Name),
			Namespace: u.jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       fmt.Sprintf("%s-ui-configuration", u.jaeger.Name),
				"app.kubernetes.io/instance":   u.jaeger.Name,
				"app.kubernetes.io/component":  "ui-configuration",
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
// support for the UI configmap if appropriate
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec, options *[]string) {
	// Check for empty map
	if jaeger.Spec.UI.Options.IsEmpty() {
		return
	}

	volume := corev1.Volume{
		Name: configurationVolumeName(jaeger),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-ui-configuration", jaeger.Name),
				},
				Items: []corev1.KeyToPath{
					corev1.KeyToPath{
						Key:  "ui",
						Path: "ui.json",
					},
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
	*options = append(*options, "--query.ui-config=/etc/config/ui.json")
}

func configurationVolumeName(jaeger *v1.Jaeger) string {
	return util.DNSName(fmt.Sprintf("%s-ui-configuration-volume", jaeger.Name))
}
