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
	if migrated, changed := upgrade1_17_0MigrateKafkaTLS(jaeger.Spec.Collector.Options); changed {
		jaeger.Spec.Collector.Options = migrated
	}

	if migrated, changed := upgrade1_17_0MigrateKafkaTLS(jaeger.Spec.Ingester.Options); changed {
		jaeger.Spec.Ingester.Options = migrated
	}

	// the common storage block also influences the collector/ingester
	if migrated, changed := upgrade1_17_0MigrateKafkaTLS(jaeger.Spec.Storage.Options); changed {
		jaeger.Spec.Storage.Options = migrated
	}

	return jaeger, nil
}

func upgrade1_17_0MigrateKafkaTLS(opts v1.Options) (v1.Options, bool) {
	optsMap := opts.GenericMap()
	changed := false
	if optsMap["kafka.consumer.authentication"] == "tls" {
		optsMap["kafka.consumer.tls.enabled"] = "true"
		changed = true
	}
	if optsMap["kafka.producer.authentication"] == "tls" {
		optsMap["kafka.producer.tls.enabled"] = "true"
		changed = true
	}
	if changed {
		return v1.NewOptions(optsMap), true
	}

	return opts, false
}
