package upgrade

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestVersionUpgradeToLatest(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}

	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.11.0" // this is the first version we have an upgrade function
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(context.Background(), cl, cl))

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, latest.v, persisted.Status.Version)
}

func TestVersionUpgradeToLatestMultinamespace(t *testing.T) {
	// prepare
	viper.Set(v1.ConfigWatchNamespace, "observability,other-observability")
	defer viper.Reset()

	nsn := types.NamespacedName{
		Name:      "my-instance",
		Namespace: "observability",
	}

	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.11.0" // this is the first version we have an upgrade function
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(context.Background(), cl, cl))

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, latest.v, persisted.Status.Version)
}

func TestVersionUpgradeToLatestOwnedResource(t *testing.T) {
	// prepare
	viper.Set(v1.ConfigIdentity, "my-identity")
	defer viper.Reset()

	nsn := types.NamespacedName{Name: "my-instance"}

	existing := v1.NewJaeger(nsn)
	existing.Labels = map[string]string{
		v1.LabelOperatedBy: viper.GetString(v1.ConfigIdentity),
	}
	existing.Status.Version = "1.11.0" // this is the first version we have an upgrade function
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(context.Background(), cl, cl))

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, latest.v, persisted.Status.Version)
}

func TestUnknownVersion(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}

	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.10.0" // we don't know how to upgrade from 1.10.0
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(context.Background(), cl, cl))

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, "1.10.0", persisted.Status.Version)
}

func TestSkipForNonOwnedInstances(t *testing.T) {
	// prepare
	viper.Set(v1.ConfigIdentity, "the-identity")
	defer viper.Reset()

	nsn := types.NamespacedName{Name: "my-instance"}

	existing := v1.NewJaeger(nsn)
	existing.Labels = map[string]string{
		v1.LabelOperatedBy: "some-other-identity",
	}
	existing.Status.Version = "1.11.0"
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1.JaegerList{})
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(context.Background(), cl, cl))

	// verify
	persisted := &v1.Jaeger{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, "1.11.0", persisted.Status.Version)
}
