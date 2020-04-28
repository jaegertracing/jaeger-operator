package upgrade

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"

	log "github.com/sirupsen/logrus"
)

func upgrade1_18_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
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

	jaeger.Spec.Collector.Options = migrateDeprecatedOptions(&jaeger, jaeger.Spec.Collector.Options, collectorDeprecatedFlags)
	jaeger.Spec.Collector.Options = transformCollectorPorts(jaeger.Logger(), jaeger.Spec.Collector.Options, collectorDeprecatedFlags)

	// Remove agent flags
	jaeger.Spec.Agent.Options = migrateAgentOptions(&jaeger)

	return jaeger, nil
}

func transformCollectorPorts(logger *log.Entry, opts v1.Options, collectorNewFlagsMap []deprecationFlagMap) v1.Options {
	// Transform port number to format :XXX
	in := opts.GenericMap()
	for _, d := range collectorNewFlagsMap {
		logger.WithFields(log.Fields{
			"from": d.from,
			"to":   d.to,
		}).Debug("flag value migrated")
		if val, exists := in[d.to]; exists {
			in[d.to] = fmt.Sprintf(":%s", val)
		}
	}

	return v1.NewOptions(in)
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
	opsMap := ops.GenericMap()

	// Removed support for tchannel, so we need to make sure grpc is enabled and properly configured.
	if _, ok := opsMap["reporter.grpc.host-port"]; !ok {
		opsMap["reporter.grpc.host-port"] = fmt.Sprintf("dns:///%s.%s:14250",
			service.GetNameForHeadlessCollectorService(jaeger), jaeger.Namespace)
	}

	return v1.NewOptions(opsMap)

}
