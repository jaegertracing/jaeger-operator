package naming

import (
	"fmt"
	jaegertracingv2 "github.com/jaegertracing/jaeger-operator/apis/jaegertracing/v2"
)

func Collector(jaeger jaegertracingv2.Jaeger) string {
	return fmt.Sprintf("%s-collector", jaeger.Name)
}
