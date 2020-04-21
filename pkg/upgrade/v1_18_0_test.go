package upgrade

import (
	"context"
	"fmt"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestUpgradeDeprecatedOptionsv1_18_0(t *testing.T) {
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := v1.NewJaeger(nsn)
	existing.Status.Version = "1.17.1"

	flagsMap := map[string]string{
		"collector.tchan-server.host-port": "collector.port",
		"collector.http-server.host-port": "collector.http-port",
		"collector.grpc-server.host-port": "collector.grpc-port",
		"collector.zipkin.host-port": "collector.zipkin.http-port",
		"admin.http.host-port": "admin-http-port"          ,
	}

	oldOptionsMap := map[string]interface{}{
		"collector.port":             "4445",
		"collector.http-port":        "8080",
		"collector.grpc-port":        "443",
		"collector.zipkin.http-port": "6544",
		"admin-http-port":            "8888",
	}

	existing.Spec.Collector.Options = v1.NewOptions(oldOptionsMap)
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
	// assert.Equal(t, latest.v, persisted.Status.Version)

	opts := persisted.Spec.Collector.Options.Map()
	for _, newFlag := range []string{
		"collector.tchan-server.host-port",
		"collector.http-server.host-port",
		"collector.grpc-server.host-port",
		"collector.zipkin.host-port",
		"admin.http.host-port" } {
		assert.Contains(t, opts,  newFlag)
		expectedValue := fmt.Sprintf(":%s", oldOptionsMap[flagsMap[newFlag]])
		assert.Equal(t, expectedValue, opts[newFlag])
	}
}
