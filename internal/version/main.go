// Package version contains the operator's version, as well as versions of underlying components.
package version

import (
	"fmt"
	"runtime"
)

var (
	version   string
	buildDate string
	jaeger    string
)

// Version holds this Operator's version as well as the version of some of the components it uses
type Version struct {
	Operator  string `json:"opentelemetry-operator"`
	BuildDate string `json:"build-date"`
	Jaeger    string `json:"jaeger-version"`
	Go        string `json:"go-version"`
}

// Get returns the Version object with the relevant information
func Get() Version {
	return Version{
		Operator:  version,
		BuildDate: buildDate,
		Jaeger:    OpenTelemetryCollector(),
		Go:        runtime.Version(),
	}
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(Operator='%v', BuildDate='%v', Jaeger='%v', Go='%v')",
		v.Operator,
		v.BuildDate,
		v.Jaeger,
		v.Go,
	)
}

// OpenTelemetryCollector returns the default OpenTelemetryCollector to use when no versions are specified via CLI or configuration
func OpenTelemetryCollector() string {
	if len(jaeger) > 0 {
		// this should always be set, as it's specified during the build
		return jaeger
	}

	// fallback value, useful for tests
	return "0.0.0"
}
