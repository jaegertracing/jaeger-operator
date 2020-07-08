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

const (
	serviceCAMountPath = "/etc/pki/ca-trust/source/service-ca"
	serviceCAFile      = "service-ca.crt"
	caBundleMountPath  = "/etc/pki/ca-trust/extracted/pem"

	// ServiceCAPath represents the in-container full path to the service-ca file
	ServiceCAPath = serviceCAMountPath + "/" + serviceCAFile
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

// GetServiceCABundle returns a service CA configmap if platform is OpenShift
func GetServiceCABundle(jaeger *v1.Jaeger) *corev1.ConfigMap {
	// Only configure the service CA if running in OpenShift
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return nil
	}

	if !deployServiceCA(jaeger) {
		jaeger.Logger().Debug("CA: Skip deploying the Jaeger instance's service CA configmap")
		return nil
	}

	jaeger.Logger().Debug("CA: Creating the service CA configmap")
	trueVar := true

	name := ServiceCAName(jaeger)
	annotations := map[string]string{
		"service.beta.openshift.io/inject-cabundle": "true",
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   jaeger.Namespace,
			Labels:      util.Labels(name, "service-ca-configmap", *jaeger),
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: jaeger.APIVersion,
				Kind:       jaeger.Kind,
				Name:       jaeger.Name,
				UID:        jaeger.UID,
				Controller: &trueVar,
			}},
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
		MountPath: caBundleMountPath,
		ReadOnly:  true,
	}

	commonSpec.Volumes = util.RemoveDuplicatedVolumes(append(commonSpec.Volumes, volume))
	commonSpec.VolumeMounts = util.RemoveDuplicatedVolumeMounts(append(commonSpec.VolumeMounts, volumeMount))
}

// AddServiceCA will modify the supplied common spec, to include
// the service CA volume and volumeMount, if running on OpenShift
func AddServiceCA(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec) {
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return
	}

	if !deployServiceCA(jaeger) {
		jaeger.Logger().Debug("CA: Skip adding the Jaeger instance's service CA volume")
		return
	}

	volume := corev1.Volume{
		Name: ServiceCAName(jaeger),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: ServiceCAName(jaeger),
				},
				Items: []corev1.KeyToPath{{
					Key:  "service-ca.crt",
					Path: "service-ca.crt",
				}},
			},
		},
	}

	volumeMount := corev1.VolumeMount{
		Name:      ServiceCAName(jaeger),
		MountPath: serviceCAMountPath,
		ReadOnly:  true,
	}

	commonSpec.Volumes = util.RemoveDuplicatedVolumes(append(commonSpec.Volumes, volume))
	commonSpec.VolumeMounts = util.RemoveDuplicatedVolumeMounts(append(commonSpec.VolumeMounts, volumeMount))
}

func deployTrustedCA(jaeger *v1.Jaeger) bool {
	for _, vm := range jaeger.Spec.JaegerCommonSpec.VolumeMounts {
		if strings.HasPrefix(vm.MountPath, caBundleMountPath) {
			// Volume Mount already exists, so don't create specific
			// one for this Jaeger instance
			return false
		}
	}
	return true
}

func deployServiceCA(jaeger *v1.Jaeger) bool {
	for _, vm := range jaeger.Spec.JaegerCommonSpec.VolumeMounts {
		if strings.HasPrefix(vm.MountPath, serviceCAMountPath) {
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

// ServiceCAName returns the name of the trusted CA
func ServiceCAName(jaeger *v1.Jaeger) string {
	return ServiceCANameFromString(jaeger.Name)
}

// ServiceCANameFromString returns the name of the trusted CA
func ServiceCANameFromString(name string) string {
	return fmt.Sprintf("%s-service-ca", name)
}
