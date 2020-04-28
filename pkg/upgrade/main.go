package upgrade

import (
	"sort"

	"github.com/Masterminds/semver"
)

func init() {
	parseSemVer()
}

func parseSemVer() {
	// ignore errors, we shouldn't have semantic version parsing errors at runtime
	semanticVersions, _ = versions(upgrades)
	startUpdatesVersion, _ = semver.NewVersion("1.11.0")
}

// Versions return the list of semantic version sorted
func versions(versions map[string]upgradeFunction) ([]*semver.Version, error) {
	versionLists := make([]*semver.Version, len(versions))
	versionIndex := 0
	for v := range versions {
		semv, err := semver.NewVersion(v)
		if err != nil {
			return nil, err
		}
		versionLists[versionIndex] = semv
		versionIndex++
	}

	// apply the updates in order
	sort.Sort(semver.Collection(versionLists))
	return versionLists, nil

}
