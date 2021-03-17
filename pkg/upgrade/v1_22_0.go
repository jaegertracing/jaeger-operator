package upgrade

import (
	"context"
	"strings"

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

	return migrateCassandraVerifyFlag(jaeger), nil
}

func flagBoolValue(v interface{}) bool {
	strValue, isString := v.(string)
	if isString {
		if strings.EqualFold(strValue, "true") {
			return true
		}
		return false
	}

	boolValue := v.(bool)
	return boolValue
}

func migrateCassandraVerifyFlag(j v1.Jaeger) v1.Jaeger {
	j.Spec.Collector.Options = updateCassandraVerifyHostFlag(j.Spec.Collector.Options)
	j.Spec.Storage.Options = updateCassandraVerifyHostFlag(j.Spec.Collector.Options)
	j.Spec.Ingress.Options = updateCassandraVerifyHostFlag(j.Spec.Collector.Options)
	return j
}

func updateCassandraVerifyHostFlag(options v1.Options) v1.Options {
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
