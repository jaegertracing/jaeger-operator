package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func upgrade1_22_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	flagMapCollector := []deprecationFlagMap{{
		from: "jaeger.tags",
		to:   "collector.tags",
	}}

	flagMapAgent := []deprecationFlagMap{{
		from: "jaeger.tags",
		to:   "agent.tags",
	}}

	j := &jaeger
	j.Spec.AllInOne.Options = migrateDeprecatedOptions(j, j.Spec.AllInOne.Options, flagMapCollector)
	j.Spec.Collector.Options = migrateDeprecatedOptions(j, j.Spec.Collector.Options, flagMapCollector)
	j.Spec.Agent.Options = migrateDeprecatedOptions(j, j.Spec.Agent.Options, flagMapAgent)

	return jaeger, nil
}
