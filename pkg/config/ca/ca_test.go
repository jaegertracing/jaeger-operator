package ca

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
)

func TestGetWithoutTrustedCA(t *testing.T) {
	// prepare
	viper.Set("platform", "other")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	trusted := GetTrustedCABundle(jaeger)
	service := GetServiceCABundle(jaeger)

	// verify
	assert.Nil(t, trusted)
	assert.Nil(t, service)
}

func TestGetWithTrustedCA(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	cm := GetTrustedCABundle(jaeger)

	// verify
	assert.NotNil(t, cm)
	assert.Equal(t, "true", cm.Labels["config.openshift.io/inject-trusted-cabundle"])
	assert.Equal(t, "", cm.Data["ca-bundle.crt"])
}

func TestGetWithServiceCA(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	cm := GetServiceCABundle(jaeger)

	// verify
	assert.NotNil(t, cm)
	assert.Equal(t, "true", cm.Annotations["service.beta.openshift.io/inject-cabundle"])
}

func TestGetWithExistingTrustedCA(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.JaegerCommonSpec.VolumeMounts = []corev1.VolumeMount{{
		MountPath: caBundleMountPath,
		Name:      "ExistingTrustedCA",
	}}

	// test
	cm := GetTrustedCABundle(jaeger)

	// verify
	assert.Nil(t, cm)
}

func TestGetWithExistingServiceCA(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.JaegerCommonSpec.VolumeMounts = []corev1.VolumeMount{{
		MountPath: serviceCAMountPath,
		Name:      "ExistingServiceCA",
	}}

	// test
	cm := GetServiceCABundle(jaeger)

	// verify
	assert.Nil(t, cm)
}

func TestUpdateWithoutCAs(t *testing.T) {
	// prepare
	viper.Set("platform", "other")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	commonSpec := v1.JaegerCommonSpec{}

	// test
	Update(jaeger, &commonSpec)
	AddServiceCA(jaeger, &commonSpec)

	// verify
	assert.Empty(t, commonSpec.Volumes)
	assert.Empty(t, commonSpec.VolumeMounts)
}

func TestUpdateWithTrustedCA(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	commonSpec := v1.JaegerCommonSpec{}

	// test
	Update(jaeger, &commonSpec)
	AddServiceCA(jaeger, &commonSpec)

	// verify
	assert.Len(t, commonSpec.Volumes, 2)
	assert.Len(t, commonSpec.VolumeMounts, 2)
}

func TestUpdateWithExistingTrustedCA(t *testing.T) {
	// prepare
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.JaegerCommonSpec.VolumeMounts = []corev1.VolumeMount{
		{
			MountPath: caBundleMountPath,
			Name:      "ExistingTrustedCA",
		},
		{
			MountPath: serviceCAMountPath,
			Name:      "ExistingServiceCA",
		},
	}
	commonSpec := v1.JaegerCommonSpec{}

	// test
	Update(jaeger, &commonSpec)
	AddServiceCA(jaeger, &commonSpec)

	// verify
	assert.Empty(t, commonSpec.Volumes)
	assert.Empty(t, commonSpec.VolumeMounts)
}
