package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func upgrade1_17_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	d := []deprecationFlagMap{{
		from: "collector.grpc.tls",
		to:   "collector.grpc.tls.enabled",
	}, {
		from: "reporter.grpc.tls",
		to:   "reporter.grpc.tls.enabled",
	}}

	return migrateAllDeprecatedOptions(jaeger, d), nil
}
