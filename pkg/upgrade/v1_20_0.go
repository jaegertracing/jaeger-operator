package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func upgrade1_20_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	d := []deprecationFlagMap{{
		from: "es.max-num-spans",
		to:   "es.max-doc-count",
	}, {
		from: "es-archive.max-num-spans",
		to:   "es-archive.max-doc-count",
	}}

	return migrateAllDeprecatedOptions(jaeger, d), nil
}
