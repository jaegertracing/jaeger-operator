package upgrade

import (
	"context"
	"testing"

	"github.com/Masterminds/semver"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func noop(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	return jaeger, nil
}

func TestVersions(t *testing.T) {
	maptoTest := map[string]upgradeFunction{
		"1.17.0": noop,
		"1.15.4": noop,
		"1.15.0": noop,
		"1.16.1": noop,
		"1.12.2": noop,
	}
	sortedSemVersions := []*semver.Version{
		semver.MustParse("1.12.2"),
		semver.MustParse("1.15.0"),
		semver.MustParse("1.15.4"),
		semver.MustParse("1.16.1"),
		semver.MustParse("1.17.0"),
	}

	semVersions, err := versions(maptoTest)
	assert.NoError(t, err)
	assert.Equal(t, semVersions, sortedSemVersions)
}

func TestVersionsError(t *testing.T) {
	maptoTest := map[string]upgradeFunction{
		"1.17.0": noop,
		"1.15.4": noop,
		"1.15.0": noop,
		"1,16.1": noop, // Bad format, coma instead of point
		"1.12.2": noop,
	}

	_, err := versions(maptoTest)
	assert.Error(t, err)
}
