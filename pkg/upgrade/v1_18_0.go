package upgrade

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
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

	agentFlags := []string{
		"collector.host-port",
		"reporter.tchannel.discovery.conn-check-timeout",
		"reporter.tchannel.discovery.min-peers",
		"reporter.tchannel.host-port",
		"reporter.tchannel.report-timeout",
	}

	jaeger.Spec.Collector.Options = migrateDeprecatedOptions(&jaeger, jaeger.Spec.Collector.Options, collectorDeprecatedFlags)
	jaeger.Spec.Collector.Options = transformCollectorPorts(&jaeger, jaeger.Spec.Collector.Options, collectorDeprecatedFlags)

	// Remove agent flags
	deleteAgentFlags := []deprecationFlagMap{}
	for _, item := range agentFlags {
		deleteAgentFlags = append(deleteAgentFlags, deprecationFlagMap{
			from: item,
			to:   "",
		})
	}

	jaeger.Spec.Agent.Options = migrateDeprecatedOptions(&jaeger, jaeger.Spec.Agent.Options, deleteAgentFlags)

	return jaeger, nil
}

func transformCollectorPorts(jaeger *v1.Jaeger, opts v1.Options, flagMap []deprecationFlagMap) v1.Options {
	// Transform port number to format :XXX
	in := opts.GenericMap()
	for _, d := range flagMap {
		jaeger.Logger().WithFields(log.Fields{
			"from": d.from,
			"to":   d.to,
		}).Debug("flag value migrated")
		if val, exists := in[d.to]; exists {
			in[d.to] = fmt.Sprintf(":%s", val)
		}
	}

	return v1.NewOptions(in)
}
