package ca

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestGetWithoutTrustedCA(t *testing.T) {
	viper.Set("platform", "other")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestGetWithoutTrustedCA"})

	cm := Get(jaeger)
	assert.Nil(t, cm)
}

func TestGetWithTrustedCA(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestGetWithTrustedCA"})

	cm := Get(jaeger)
	assert.NotNil(t, cm)
	assert.Equal(t, "true", cm.Labels["config.openshift.io/inject-trusted-cabundle"])
	assert.Equal(t, "", cm.Data["ca-bundle.crt"])
}

func TestUpdateWithoutTrustedCA(t *testing.T) {
	viper.Set("platform", "other")
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateWithoutTrustedCA"})

	commonSpec := v1.JaegerCommonSpec{}

	Update(jaeger, &commonSpec)
	assert.Len(t, commonSpec.Volumes, 0)
}

func TestUpdateWithTrustedCA(t *testing.T) {
	viper.Set("platform", v1.FlagPlatformOpenShift)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateWithTrustedCA"})

	commonSpec := v1.JaegerCommonSpec{}

	Update(jaeger, &commonSpec)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Equal(t, commonSpec.Volumes[0].Name, TrustedCAName(jaeger))
}
