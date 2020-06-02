package ca

import (
	"fmt"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// Get returns a trusted CA bundle configmap if platform is OpenShift
func Get(jaeger *v1.Jaeger) *corev1.ConfigMap {
	// Only configure the trusted CA if running in OpenShift
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return nil
	}

	jaeger.Logger().Debug("CA: Creating the trustedCABundle configmap")
	trueVar := true

	// See https://docs.openshift.com/container-platform/4.4/networking/configuring-a-custom-pki.html#certificate-injection-using-operators_configuring-a-custom-pki
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TrustedCAName(jaeger),
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"config.openshift.io/inject-trusted-cabundle": "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: jaeger.APIVersion,
					Kind:       jaeger.Kind,
					Name:       jaeger.Name,
					UID:        jaeger.UID,
					Controller: &trueVar,
				},
			},
		},
		Data: map[string]string{
			"ca-bundle.crt": "",
		},
	}
}

// Update will modify the supplied common spec, to include
// trusted CA bundle volume, if running on OpenShift
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec) {
	// Only configure the trusted CA if running in OpenShift
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return
	}

	volume := corev1.Volume{
		Name: TrustedCAName(jaeger),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: TrustedCAName(jaeger),
				},
				Items: []corev1.KeyToPath{
					corev1.KeyToPath{
						Key:  "ca-bundle.crt",
						Path: "tls-ca-bundle.pem",
					},
				},
			},
		},
	}
	commonSpec.Volumes = append(commonSpec.Volumes, volume)
}

// TrustedCAName returns the name of the trusted CA
func TrustedCAName(jaeger *v1.Jaeger) string {
	return fmt.Sprintf("%s-trusted-ca", jaeger.Name)
}
