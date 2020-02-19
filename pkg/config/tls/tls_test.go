package tls

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestNoTLSConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestNoTLSConfig"})

	config := NewConfig(jaeger)
	cm := config.Get()
	assert.NotNil(t, cm)
}

func TestUpdateWithTLSConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateWithTLSConfig"})
	viper.Set("platform", v1.FlagPlatformOpenShift)

	commonSpec := v1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Len(t, options, 2)
	assert.Equal(t, "--collector.grpc.tls=true", options[0])
	assert.Equal(t, "--collector.grpc.tls.cert=/etc/config/service-ca.crt", options[1])
}
