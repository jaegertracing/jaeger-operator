package ca

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// GetTrustedCABundle returns a trusted CA bundle configmap if platform is OpenShift
func GetTrustedCABundle(jaeger *v1.Jaeger) *corev1.ConfigMap {
	// Only configure the trusted CA if running in OpenShift
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return nil
	}

	if !deployTrustedCA(jaeger) {
		jaeger.Logger().Debug("CA: Skip deploying the Jaeger instance's trustedCABundle configmap")
		return nil
	}

	jaeger.Logger().Debug("CA: Creating the trustedCABundle configmap")
	trueVar := true

	name := TrustedCAName(jaeger)
	labels := util.Labels(name, "ca-configmap", *jaeger)
	labels["config.openshift.io/inject-trusted-cabundle"] = "true"

	// See https://docs.openshift.com/container-platform/4.4/networking/configuring-a-custom-pki.html#certificate-injection-using-operators_configuring-a-custom-pki
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: jaeger.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: jaeger.APIVersion,
				Kind:       jaeger.Kind,
				Name:       jaeger.Name,
				UID:        jaeger.UID,
				Controller: &trueVar,
			}},
		},
		Data: map[string]string{
			"ca-bundle.crt": "",
		},
	}
}

// Update will modify the supplied common spec, to include
// trusted CA bundle volume and volumeMount, if running on OpenShift
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec) {
	// Only configure the trusted CA if running in OpenShift
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return
	}

	if !deployTrustedCA(jaeger) {
		jaeger.Logger().Debug("CA: Skip adding the Jaeger instance's trustedCABundle volume")
		return
	}

	volume := corev1.Volume{
		Name: TrustedCAName(jaeger),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: TrustedCAName(jaeger),
				},
				Items: []corev1.KeyToPath{{
					Key:  "ca-bundle.crt",
					Path: "tls-ca-bundle.pem",
				}},
			},
		},
	}

	volumeMount := corev1.VolumeMount{
		Name:      TrustedCAName(jaeger),
		MountPath: "/etc/pki/ca-trust/extracted/pem",
		ReadOnly:  true,
	}

	commonSpec.Volumes = util.RemoveDuplicatedVolumes(append(commonSpec.Volumes, volume))
	commonSpec.VolumeMounts = util.RemoveDuplicatedVolumeMounts(append(commonSpec.VolumeMounts, volumeMount))
}

func deployTrustedCA(jaeger *v1.Jaeger) bool {
	for _, vm := range jaeger.Spec.JaegerCommonSpec.VolumeMounts {
		if strings.HasPrefix(vm.MountPath, "/etc/pki/ca-trust/extracted/pem") {
			// Volume Mount already exists, so don't create specific
			// one for this Jaeger instance
			return false
		}
	}
	return true
}

// TrustedCAName returns the name of the trusted CA
func TrustedCAName(jaeger *v1.Jaeger) string {
	return TrustedCANameFromString(jaeger.Name)
}

// TrustedCANameFromString returns the name of the trusted CA
func TrustedCANameFromString(name string) string {
	return fmt.Sprintf("%s-trusted-ca", name)
}
