package upgrade

import (
	log "github.com/sirupsen/logrus"

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

func migrateAllDeprecatedOptions(jaeger v1.Jaeger, flagMap []deprecationFlagMap) v1.Jaeger {
	j := &jaeger
	j.Spec.AllInOne.Options = migrateDeprecatedOptions(j, j.Spec.AllInOne.Options, flagMap)
	j.Spec.Collector.Options = migrateDeprecatedOptions(j, j.Spec.Collector.Options, flagMap)
	j.Spec.Query.Options = migrateDeprecatedOptions(j, j.Spec.Query.Options, flagMap)
	j.Spec.Agent.Options = migrateDeprecatedOptions(j, j.Spec.Agent.Options, flagMap)
	j.Spec.Ingester.Options = migrateDeprecatedOptions(j, j.Spec.Ingester.Options, flagMap)
	j.Spec.Storage.Options = migrateDeprecatedOptions(j, j.Spec.Storage.Options, flagMap)

	return jaeger
}

func migrateDeprecatedOptions(jaeger *v1.Jaeger, opts v1.Options, flagMap []deprecationFlagMap) v1.Options {
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
