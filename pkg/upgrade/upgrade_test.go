package upgrade

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	opver "github.com/jaegertracing/jaeger-operator/pkg/version"
)

func TestVersionUpgradeToLatest(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}

	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.11.0" // this is the first version we have an upgrade function
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	// test
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, "1.12.0"))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, "1.12.0", persisted.Status.Version)
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
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	// test
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, "1.12.0"))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, "1.12.0", persisted.Status.Version)
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
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	// test
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, "1.12.0"))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, "1.12.0", persisted.Status.Version)
}

func TestUnknownVersion(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}

	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.10.0" // we don't know how to upgrade from 1.10.0
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	// test
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, "1.12.0"))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
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
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	// test
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, opver.Get().Jaeger))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, "1.11.0", persisted.Status.Version)
}

func TestErrorForInvalidSemVer(t *testing.T) {
	invalidVersion := "xxx...xx"
	testUpdates := map[string]upgradeFunction{}
	for k, v := range upgrades {
		testUpdates[k] = v
	}
	testUpdates[invalidVersion] = func(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
		return jaeger, nil
	}
	_, err := versions(testUpdates)
	// test
	require.Error(t, err)
}

func TestSkipUpgradeForVersionsGreaterThanLatest(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}

	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "999.999"
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	// test
	require.NoError(t, ManagedInstances(context.Background(), cl, cl, opver.Get().Jaeger))

	// verify
	persisted := &v1.Jaeger{}
	require.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, existing.Status.Version, persisted.Status.Version)
}

func TestVersionMapIsValid(t *testing.T) {
	_, err := versions(upgrades)
	require.NoError(t, err)
}
