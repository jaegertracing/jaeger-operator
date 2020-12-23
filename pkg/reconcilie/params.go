package reconcilie

import (
	"github.com/go-logr/logr"
	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Params struct {
	Client   client.Client
	Instance jaegertracingv2.Jaeger
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Strategy strategy.Strategy
}
