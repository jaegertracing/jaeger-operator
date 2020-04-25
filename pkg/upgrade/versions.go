package upgrade

import (
	"context"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Masterminds/semver"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

type upgradeFunction = func(ctx context.Context, client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error)

var (
	semanticVersions []*semver.Version

	upgrades = map[string]upgradeFunction{
		"1.15.0": upgrade1_15_0,
		"1.17.0": upgrade1_17_0,
	}
)

// Versions return the list of semantic version sorted
func Versions() ([]*semver.Version, error) {

	if semanticVersions != nil {
		return semanticVersions, nil
	}

	versionLists := make([]*semver.Version, len(upgrades))
	versionIndex := 0
	for v := range upgrades {
		semv, err := semver.NewVersion(v)
		if err != nil {
			return nil, err
		}
		versionLists[versionIndex] = semv
		versionIndex++
	}

	// apply the updates in order
	sort.Sort(semver.Collection(versionLists))
	semanticVersions = versionLists
	return semanticVersions, nil

}
