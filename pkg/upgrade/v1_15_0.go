package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func upgrade1_15_0(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	d := []deprecationFlagMap{{
		from: "collector.grpc.tls.client.ca", // client dot ca
		to:   "collector.grpc.tls.client-ca", // client dash ca
	}, {
		from: "collector.host-port",
		to:   "reporter.tchannel.host-port",
	}, {
		from: "discovery.conn-check-timeout",
		to:   "reporter.tchannel.discovery.conn-check-timeout",
	}, {
		from: "discovery.min-peers",
		to:   "reporter.tchannel.discovery.min-peers",
	}, {
		from: "health-check-http-port",
		to:   "admin-http-port",
	}, {
		from: "cassandra-archive.enable-dependencies-v2",
		to:   "",
	}, {
		from: "cassandra.enable-dependencies-v2",
		to:   "",
	}}

	return migrateAllDeprecatedOptions(jaeger, d), nil
}
