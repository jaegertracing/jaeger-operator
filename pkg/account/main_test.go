package account

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

func TestWithSecurityNil(t *testing.T) {
	jaeger := v1.NewJaeger("TestWithOAuthProxyNil")
	assert.Equal(t, v1.IngressSecurityNone, jaeger.Spec.Ingress.Security)
	sas := Get(jaeger)
	assert.Len(t, sas, 1)
	assert.Equal(t, getMain(jaeger), sas[0])
}

func TestWithSecurityNone(t *testing.T) {
	jaeger := v1.NewJaeger("TestWithOAuthProxyFalse")
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNone
	sas := Get(jaeger)
	assert.Len(t, sas, 1)
	assert.Equal(t, getMain(jaeger), sas[0])
}

func TestWithSecurityOAuthProxy(t *testing.T) {
	jaeger := v1.NewJaeger("TestWithOAuthProxyTrue")
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy

	assert.Len(t, Get(jaeger), 2)
}

func TestJaegerName(t *testing.T) {
	jaeger := v1.NewJaeger("foo")
	jaeger.Spec.ServiceAccount = "bar"
	jaeger.Spec.Collector.ServiceAccount = "col-sa"
	jaeger.Spec.Query.ServiceAccount = "query-sa"
	jaeger.Spec.Agent.ServiceAccount = "agent-sa"
	jaeger.Spec.AllInOne.ServiceAccount = "aio-sa"

	assert.Equal(t, "foo", JaegerServiceAccountFor(jaeger, ""))
	assert.Equal(t, "col-sa", JaegerServiceAccountFor(jaeger, util.CollectorComponent))
	assert.Equal(t, "query-sa", JaegerServiceAccountFor(jaeger, util.QueryComponent))
	assert.Equal(t, "aio-sa", JaegerServiceAccountFor(jaeger, util.AllInOneComponent))
	assert.Equal(t, "agent-sa", JaegerServiceAccountFor(jaeger, util.AgentComponent))
	assert.Equal(t, "bar", JaegerServiceAccountFor(jaeger, util.IngesterComponent))
}
