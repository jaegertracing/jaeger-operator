package jaeger

import (
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/upgrade"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

func (r *ReconcileJaeger) applyUpgrades(jaeger v1.Jaeger) (v1.Jaeger, error) {
	currentVersions := version.Get()

	if len(jaeger.Status.Version) > 0 {
		if jaeger.Status.Version != currentVersions.Jaeger {
			// in theory, the version from the Status could be higher than currentVersions.Jaeger, but we let the upgrade routine
			// check/handle it
			upgraded, err := upgrade.ManagedInstance(r.client, jaeger)
			if err != nil {
				return jaeger, err
			}
			jaeger = upgraded
		}
	}

	// at this point, the Jaeger we are managing is in sync with the Operator's version
	// if this is a new object, no upgrade was made, so, we just set the version
	jaeger.Status.Version = version.Get().Jaeger
	return jaeger, nil
}
