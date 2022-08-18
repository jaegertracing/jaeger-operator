package upgrade

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
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

	flagMapQuery := []deprecationFlagMap{
		{
			from: "downsampling.hashsalt",
			to:   "",
		},
		{
			from: "downsampling.ratio",
			to:   "",
		},
	}

	j := &jaeger
	j.Spec.AllInOne.Options = migrateDeprecatedOptions(j, j.Spec.AllInOne.Options, flagMapCollector)
	j.Spec.Collector.Options = migrateDeprecatedOptions(j, j.Spec.Collector.Options, flagMapCollector)
	j.Spec.Agent.Options = migrateDeprecatedOptions(j, j.Spec.Agent.Options, flagMapAgent)
	j.Spec.Query.Options = migrateDeprecatedOptions(j, j.Spec.Query.Options, flagMapQuery)

	// Migrate query host/port flags
	j.Spec.Query.Options = migrateQueryHostPortFlagsv1_22_0(j.Spec.Query.Options)

	return migrateCassandraVerifyFlagv1_22_0(jaeger), nil
}

func migrateCassandraVerifyFlagv1_22_0(j v1.Jaeger) v1.Jaeger {
	j.Spec.Collector.Options = updateCassandraVerifyHostFlagv1_22_0(j.Spec.Collector.Options)
	j.Spec.Storage.Options = updateCassandraVerifyHostFlagv1_22_0(j.Spec.Storage.Options)
	j.Spec.Ingress.Options = updateCassandraVerifyHostFlagv1_22_0(j.Spec.Ingress.Options)
	return j
}

func updateCassandraVerifyHostFlagv1_22_0(options v1.Options) v1.Options {
	oldFlag := "cassandra.tls.verify-host"
	newFlag := "cassandra.tls.skip-host-verify"

	in := options.GenericMap()
	if oldFlagValue, exist := in[oldFlag]; exist {
		delete(in, oldFlag)
		if !flagBoolValue(oldFlagValue) {
			in[newFlag] = "true"
		}
	}
	return v1.NewOptions(in)
}

func migrateQueryHostPortFlagsv1_22_0(options v1.Options) v1.Options {
	in := options.GenericMap()

	port, hasPortFlag := in["query.port"]
	hostPort, hasHostPortFlag := in["query.host-port"]

	_, hasGrpcFlag := in["query.grpc-server.host-port"]
	_, hasHTTPFlag := in["query.http-server.host-port"]

	newValue := ""
	if hasPortFlag {
		newValue = ":" + port.(string)
	} else if hasHostPortFlag {
		newValue = hostPort.(string)
	}

	if newValue != "" {
		if !hasGrpcFlag {
			in["query.grpc-server.host-port"] = newValue
		}
		if !hasHTTPFlag {
			in["query.http-server.host-port"] = newValue
		}
	}

	delete(in, "query.port")
	delete(in, "query.host-port")

	return v1.NewOptions(in)
}
