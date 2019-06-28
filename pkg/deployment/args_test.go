package deployment

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestArgs(t *testing.T) {
	// prepare
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "TestArgs"})
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"memory.max-traces": 10000})
	jaeger.Spec.AllInOne.Options = v1.NewOptions(map[string]interface{}{"collector.http-port": 14268})

	// test
	args := allArgs(jaeger.Spec.Storage.Options, jaeger.Spec.AllInOne.Options)

	// verify
	sort.Strings(args)
	assert.Equal(t, "--collector.http-port=14268", args[0])
	assert.Equal(t, "--memory.max-traces=10000", args[1])
}
