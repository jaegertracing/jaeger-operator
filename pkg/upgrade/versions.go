package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type version struct {
	v       string
	upgrade func(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error)
}

var (
	v1_15_0 = version{v: "1.15.0", upgrade: upgrade1_15_0}
	v1_17_0 = version{v: "1.17.0", upgrade: upgrade1_17_0}

	versions = map[string]version{
		v1_15_0.v: v1_15_0,
		v1_17_0.v: v1_17_0,
	}
)

func noop(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	return jaeger, nil
}