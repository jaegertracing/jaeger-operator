package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func Test1_28_upgrade(t *testing.T) {
	j := v1.Jaeger{}
	j, err := upgrade1_28_0(context.Background(), nil, j)
	require.NoError(t, err)
	assert.Equal(t, " ", *j.Spec.Ingress.Openshift.SAR)
}
