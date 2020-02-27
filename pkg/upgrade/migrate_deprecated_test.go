package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestMigrateAllDeprecated(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := *v1.NewJaeger(nsn)
	existing.Spec.Collector.Options = v1.NewOptions(map[string]interface{}{
		"migrate-from":  "value-to-migrate",
		"to-be-removed": "value-will-disappear",
	})

	d := []deprecationFlagMap{{
		from: "migrate-from",
		to:   "migrate-to",
	}, {
		from: "to-be-removed",
		to:   "",
	}}

	// test
	upgraded := migrateAllDeprecatedOptions(existing, d)

	// verify
	opts := upgraded.Spec.Collector.Options.Map()
	assert.Contains(t, opts, "migrate-to")
	assert.Equal(t, "value-to-migrate", opts["migrate-to"])
	assert.NotContains(t, opts, "migrate-from")
	assert.NotContains(t, opts, "to-be-removed")
}
