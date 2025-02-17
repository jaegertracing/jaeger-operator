package tls

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
)

func TestUpdateWithTLSSecret(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestUpdateWithTLSSecret"})
	autodetect.OperatorConfiguration.SetPlatform(autodetect.OpenShiftPlatform)

	commonSpec := v1.JaegerCommonSpec{}
	options := []string{}

	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.Volumes, 1)
	assert.Len(t, commonSpec.VolumeMounts, 1)
	assert.Len(t, options, 3)
	assert.Equal(t, "--collector.grpc.tls.enabled=true", options[0])
	assert.Equal(t, "--collector.grpc.tls.cert=/etc/tls-config/tls.crt", options[1])
	assert.Equal(t, "--collector.grpc.tls.key=/etc/tls-config/tls.key", options[2])
}
