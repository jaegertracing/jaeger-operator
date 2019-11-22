package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func upgrade1_14_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	// this has the same content as `noop`, but it's added a separate function
	// to serve as template for versions with an actual upgrade procedure
	return jaeger, nil
}
