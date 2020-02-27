package upgrade

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func upgrade1_17_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	prefix := []string{
		"collector.grpc",
		"reporter.grpc",
		"es",
		"es-archive",
		"cassandra",
		"cassandra-archive",
		"kafka.consumer", // the option kafka.consumer.tls never really existed, but it's there now, and in deprecated mode...
		"kafka.producer", // same as the above comment
	}
	d := []deprecationFlagMap{}

	for _, item := range prefix {
		d = append(d, deprecationFlagMap{
			from: fmt.Sprintf("%s.tls", item),
			to:   fmt.Sprintf("%s.tls.enabled", item),
		})
	}

	jaeger = migrateAllDeprecatedOptions(jaeger, d)

	// for the collector and ingester, if we have TLS options but not ".enabled", we add the ".enabled" option
	collectorOpts := jaeger.Spec.Collector.Options.GenericMap()
	if collectorOpts["kafka.producer.authentication"] == "tls" {
		collectorOpts["kafka.producer.tls.enabled"] = "true"
		jaeger.Spec.Collector.Options = v1.NewOptions(collectorOpts)
	}

	ingesterOpts := jaeger.Spec.Ingester.Options.GenericMap()
	changed := false
	if ingesterOpts["kafka.consumer.authentication"] == "tls" {
		ingesterOpts["kafka.consumer.tls.enabled"] = "true"
		changed = true
	}
	if ingesterOpts["kafka.producer.authentication"] == "tls" {
		ingesterOpts["kafka.producer.tls.enabled"] = "true"
		changed = true
	}
	if changed {
		jaeger.Spec.Ingester.Options = v1.NewOptions(ingesterOpts)
	}

	return jaeger, nil
}
