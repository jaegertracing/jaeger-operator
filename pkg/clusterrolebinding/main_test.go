package clusterrolebinding

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestGetClusterRoleBinding(t *testing.T) {
	// prepare
	name := "TestGetClusterRoleBinding"
	trueVar := true

	viper.Set("auth-delegator-available", true)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &trueVar
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.OpenShift.DelegateURLs = `{"/":{"namespace": "default", "resource": "pods", "verb": "get"}}`

	// test
	crbs := Get(jaeger)

	// verify
	assert.Len(t, crbs, 1)
	assert.Equal(t, "system:auth-delegator", crbs[0].RoleRef.Name)
	assert.Equal(t, "ClusterRole", crbs[0].RoleRef.Kind)

	assert.Len(t, crbs[0].Subjects, 1)
	assert.Equal(t, account.OAuthProxyAccountNameFor(jaeger), crbs[0].Subjects[0].Name)
	assert.Equal(t, "ServiceAccount", crbs[0].Subjects[0].Kind)
	assert.Len(t, crbs[0].Subjects[0].Namespace, 0) // cluster roles aren't namespaced
}

func TestIngressDisabled(t *testing.T) {
	// prepare
	name := "TestIngressDisabled"
	falseVar := false

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &falseVar
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNone
	jaeger.Spec.Ingress.OpenShift.DelegateURLs = `{"/":{"namespace": "default", "resource": "pods", "verb": "get"}}`

	// test
	crbs := Get(jaeger)

	// verify
	assert.Len(t, crbs, 0)
}

func TestNotOAuthProxy(t *testing.T) {
	// prepare
	name := "TestNotOAuthProxy"
	trueVar := true

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &trueVar
	jaeger.Spec.Ingress.Security = v1.IngressSecurityNone
	jaeger.Spec.Ingress.OpenShift.DelegateURLs = `{"/":{"namespace": "default", "resource": "pods", "verb": "get"}}`

	// test
	crbs := Get(jaeger)

	// verify
	assert.Len(t, crbs, 0)
}

func TestAuthDelegatorNotAvailable(t *testing.T) {
	// prepare
	name := "TestAuthDelegatorNotAvailable"
	trueVar := true

	viper.Set("auth-delegator-available", false)
	defer viper.Reset()

	jaeger := v1.NewJaeger(types.NamespacedName{Name: name})
	jaeger.Spec.Ingress.Enabled = &trueVar
	jaeger.Spec.Ingress.Security = v1.IngressSecurityOAuthProxy
	jaeger.Spec.Ingress.OpenShift.DelegateURLs = `{"/":{"namespace": "default", "resource": "pods", "verb": "get"}}`

	// test
	crbs := Get(jaeger)

	// verify
	assert.Len(t, crbs, 0)
}
