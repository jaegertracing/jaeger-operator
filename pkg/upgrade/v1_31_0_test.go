package upgrade

import (
	"context"
	"testing"

	esv1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpgradeJaegerv1_31_0(t *testing.T) {
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.30.0"
	existing.Spec.Storage.Type = v1.JaegerESStorage

	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1.GroupVersion, &v1.Jaeger{})
	s.AddKnownTypes(v1.GroupVersion, &v1.JaegerList{})

	// Should return an error related to missing type (because we haven't added ES type to schema)
	cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
	_, err := upgrade1_31_0(context.Background(), cl, *existing)
	require.Error(t, err)

	// Should not return an error, ven if the ES instance doesn't exist.
	s.AddKnownTypes(esv1.GroupVersion, &esv1.Elasticsearch{})
	_, err = upgrade1_31_0(context.Background(), cl, *existing)
	require.NoError(t, err)
}
