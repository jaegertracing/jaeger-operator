package upgrade

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"

	"github.com/go-logr/logr"
)

func upgrade1_18_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	// Transform collector flags
	jaeger.Spec.Collector.Options = migrateCollectorOptions(&jaeger)
	// Remove agent flags
	jaeger.Spec.Agent.Options = migrateAgentOptions(&jaeger)

	return jaeger, nil
}

func migrateCollectorOptions(jaeger *v1.Jaeger) v1.Options {
	collectorDeprecatedFlags := []deprecationFlagMap{
		{
			from: "collector.port",
			to:   "collector.tchan-server.host-port",
		},
		{
			from: "collector.http-port",
			to:   "collector.http-server.host-port",
		},
		{
			from: "collector.grpc-port",
			to:   "collector.grpc-server.host-port",
		},
		{
			from: "collector.zipkin.http-port",
			to:   "collector.zipkin.host-port",
		},
		{
			from: "admin-http-port",
			to:   "admin.http.host-port",
		},
	}
	opts := migrateDeprecatedOptions(jaeger, jaeger.Spec.Collector.Options, collectorDeprecatedFlags)
	return transformCollectorPorts(jaeger.Logger(), opts, collectorDeprecatedFlags)
}

func migrateAgentOptions(jaeger *v1.Jaeger) v1.Options {
	deleteAgentFlags := []deprecationFlagMap{
		{from: "collector.host-port"},
		{from: "reporter.tchannel.discovery.conn-check-timeout"},
		{from: "reporter.tchannel.discovery.min-peers"},
		{from: "reporter.tchannel.host-port"},
		{from: "reporter.tchannel.report-timeout"},
	}

	ops := migrateDeprecatedOptions(jaeger, jaeger.Spec.Agent.Options, deleteAgentFlags)
	return v1.NewOptions(ops.GenericMap())
}

func transformCollectorPorts(logger logr.Logger, opts v1.Options, collectorNewFlagsMap []deprecationFlagMap) v1.Options {
	// Transform port number to format :XXX
	in := opts.GenericMap()
	for _, d := range collectorNewFlagsMap {
		logger.V(-1).Info(
			"flag value migrated",
			"from", d.from,
			"to", d.to,
		)
		if val, exists := in[d.to]; exists {
			in[d.to] = fmt.Sprintf(":%s", val)
		}
	}
	return v1.NewOptions(in)
}
