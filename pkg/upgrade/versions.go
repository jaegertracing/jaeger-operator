package upgrade

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type version struct {
	v       string
	upgrade func(client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error)
	next    *version
}

var (
	v1_11_0 = version{v: "1.11.0", upgrade: noop, next: &v1_12_0}
	v1_12_0 = version{v: "1.12.0", upgrade: noop, next: &v1_13_0}
	v1_13_0 = version{v: "1.13.0", upgrade: noop, next: &v1_13_1}
	v1_13_1 = version{v: "1.13.1", upgrade: noop, next: &v1_14_0}
	v1_14_0 = version{v: "1.14.0", upgrade: upgrade1_14_0, next: &v1_15_0}
	v1_15_0 = version{v: "1.15.0", upgrade: upgrade1_15_0}

	latest = &v1_15_0

	versions = map[string]version{
		v1_11_0.v: v1_11_0,
		v1_12_0.v: v1_12_0,
		v1_13_0.v: v1_13_0,
		v1_13_1.v: v1_13_1,
		v1_14_0.v: v1_14_0,
		v1_15_0.v: v1_15_0,
	}
)

func noop(client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	return jaeger, nil
}
