package jaeger

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestDirectNextMinor(t *testing.T) {
	viper.Set("jaeger-version", "")
	defer viper.Reset()

	// prepare
	nsn := types.NamespacedName{
		Name: "my-instance",
	}

	r := &ReconcileJaeger{}
	j := *v1.NewJaeger(nsn)
	j.Status.Version = "1.12.0"

	// test
	j, err := r.applyUpgrades(context.Background(), j)

	// verify
	require.NoError(t, err)

	// we cannot make any other assumptions here, but we know that 1.12.0 is an older
	// version, so, at least the status field should have been updated
	assert.NotEqual(t, "1.12.0", j.Status.Version)
}

func TestSetVersionOnNewInstance(t *testing.T) {
	// prepare
	r := &ReconcileJaeger{}
	j := *v1.NewJaeger(types.NamespacedName{Name: "my-instance"})

	// test
	j, err := r.applyUpgrades(context.Background(), j)

	// verify
	require.NoError(t, err)

	// we cannot make any other assumptions here, but we know that 1.12.0 is an older
	// version, so, at least the status field should have been updated
	assert.NotEmpty(t, j.Status.Version)
}
