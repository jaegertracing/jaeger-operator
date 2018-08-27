package deployment

import (
	"sort"
	"testing"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestArgs(t *testing.T) {
	// prepare
	jaeger := v1alpha1.NewJaeger("TestArgs")
	jaeger.Spec.Storage.Options = v1alpha1.NewOptions(map[string]interface{}{"memory.max-traces": 10000})
	jaeger.Spec.AllInOne.Options = v1alpha1.NewOptions(map[string]interface{}{"collector.http-port": 14268})

	// test
	args := allArgs(jaeger.Spec.Storage.Options, jaeger.Spec.AllInOne.Options)

	// verify
	sort.Strings(args)
	assert.Equal(t, "--collector.http-port=14268", args[0])
	assert.Equal(t, "--memory.max-traces=10000", args[1])
}
