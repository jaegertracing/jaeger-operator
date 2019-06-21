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
	v1_13_1 = version{v: "1.13.1", upgrade: upgrade1_13_1}

	latest = &v1_13_1

	versions = map[string]version{
		v1_11_0.v: v1_11_0,
		v1_12_0.v: v1_12_0,
		v1_13_0.v: v1_13_0,
		v1_13_1.v: v1_13_1,
	}
)

func noop(client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	return jaeger, nil
}
