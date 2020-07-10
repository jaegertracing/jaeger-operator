package ingress

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	extv1beta "k8s.io/api/extensions/v1beta1"
	netv1beta "k8s.io/api/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func getIngress() *netv1beta.Ingress {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestQueryIngressTLSSecret"})
	jaeger.Spec.Ingress.TLS = []v1.JaegerIngressTLSSpec{{
		SecretName: "test-secret",
	}}

	ingress := NewQueryIngress(jaeger)
	dep := ingress.Get()
	return dep
}

func TestIngressNetworkingAPI(t *testing.T) {
	viper.Set("ingress-api", NetworkingAPI)
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient()

	ingressClient := NewIngressClient(cl, cl)
	ingress := getIngress()
	// Test create
	err := ingressClient.Create(context.Background(), ingress)
	assert.NoError(t, err)

	// Get ingress for both APIs
	netIngress := &netv1beta.Ingress{}
	extIngress := &extv1beta.Ingress{}

	// extension.Ingress should not exist
	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, extIngress)
	require.True(t, apierrors.IsNotFound(err))

	// networking.Ingress
	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, netIngress)
	assert.NoError(t, err)

	// List
	netIngressList := &netv1beta.IngressList{}
	err = ingressClient.List(context.Background(), netIngressList)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(netIngressList.Items))

	// Should return 0 (not using extension API)
	extIngressList := &extv1beta.IngressList{}
	err = cl.List(context.Background(), extIngressList)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(extIngressList.Items))

	// Update
	updatedIngress := &netv1beta.Ingress{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, updatedIngress)
	require.NoError(t, err)

	updatedIngress.Spec.Backend.ServiceName = "updated-srv"
	err = ingressClient.Update(context.Background(), updatedIngress)
	require.NoError(t, err)

	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, netIngress)
	require.NoError(t, err)
	assert.Equal(t, "updated-srv", netIngress.Spec.Backend.ServiceName)

	// Delete
	err = ingressClient.Delete(context.Background(), ingress)
	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, netIngress)
	require.Error(t, err)
	assert.Equal(t, err.(*apierrors.StatusError).ErrStatus.Reason, metav1.StatusReasonNotFound)
}

func TestIngressExtensionAPI(t *testing.T) {
	viper.Set("ingress-api", ExtensionAPI)
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient()

	ingressClient := NewIngressClient(cl, cl)
	ingress := getIngress()
	// Test create
	err := ingressClient.Create(context.Background(), ingress)
	assert.NoError(t, err)

	// Get ingress for both APIs
	netIngress := &netv1beta.Ingress{}
	extIngress := &extv1beta.Ingress{}

	// networking.Ingress should not exist
	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, netIngress)
	require.Error(t, err)
	assert.Equal(t, err.(*apierrors.StatusError).ErrStatus.Reason, metav1.StatusReasonNotFound)

	// extension.Ingress
	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, extIngress)
	assert.NoError(t, err)

	// List
	extIngressList := &netv1beta.IngressList{}
	err = ingressClient.List(context.Background(), extIngressList)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(extIngressList.Items))

	// Should return 0 (not using networking API)
	netIngressList := &netv1beta.IngressList{}
	err = cl.List(context.Background(), netIngressList)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(netIngressList.Items))

	// Update
	updatedIngressExt := &extv1beta.Ingress{}

	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, updatedIngressExt)
	require.NoError(t, err)
	updatedIngress := ingressClient.fromExtToNet(*updatedIngressExt)

	updatedIngress.Spec.Backend.ServiceName = "updated-srv"
	err = ingressClient.Update(context.Background(), &updatedIngress)
	require.NoError(t, err)

	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, extIngress)
	require.NoError(t, err)
	assert.Equal(t, "updated-srv", extIngress.Spec.Backend.ServiceName)

	// Delete
	err = ingressClient.Delete(context.Background(), ingress)
	err = cl.Get(context.Background(), types.NamespacedName{Name: ingress.GetName(), Namespace: ingress.GetNamespace()}, extIngress)
	require.Error(t, err)
	assert.Equal(t, err.(*apierrors.StatusError).ErrStatus.Reason, metav1.StatusReasonNotFound)
}
