package upgrade

import (
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// deprecationFlagMap is a map from deprecated flags to their new variants
// this should be used when there's a 1-1 mapping between the old and new values
// or when there's no need for a new value, which effectively just removes the
// deprecated without replacement
type deprecationFlagMap struct {
	from string
	to   string
}

func upgrade1_15_0(client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
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

	j := &jaeger
	j.Spec.AllInOne.Options = v1_15_0MigrateDeprecatedOptions(j, j.Spec.AllInOne.Options, d)
	j.Spec.Collector.Options = v1_15_0MigrateDeprecatedOptions(j, j.Spec.Collector.Options, d)
	j.Spec.Query.Options = v1_15_0MigrateDeprecatedOptions(j, j.Spec.Query.Options, d)
	j.Spec.Agent.Options = v1_15_0MigrateDeprecatedOptions(j, j.Spec.Agent.Options, d)
	j.Spec.Ingester.Options = v1_15_0MigrateDeprecatedOptions(j, j.Spec.Ingester.Options, d)
	j.Spec.Storage.Options = v1_15_0MigrateDeprecatedOptions(j, j.Spec.Storage.Options, d)

	return jaeger, nil
}

func v1_15_0MigrateDeprecatedOptions(jaeger *v1.Jaeger, opts v1.Options, flagMap []deprecationFlagMap) v1.Options {
	in := opts.GenericMap()
	for _, d := range flagMap {
		if val, exists := in[d.from]; exists {
			jaeger.Logger().WithFields(log.Fields{
				"from": d.from,
				"to":   d.to,
			}).Debug("flag migrated")

			// if the new flag is "", there's no replacement, just skip and delete the old value
			if len(d.to) > 0 {
				in[d.to] = val
			}
			delete(in, d.from)
		}
	}

	return v1.NewOptions(in)
}
