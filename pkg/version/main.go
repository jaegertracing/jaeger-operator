package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	version       string
	buildDate     string
	defaultJaeger string
	defaultAgent  string
)

// Version holds this Operator's version as well as the version of some of the components it uses
type Version struct {
	Operator  string `json:"jaeger-operator"`
	BuildDate string `json:"build-date"`
	Jaeger    string `json:"jaeger-version"`
	Agent     string `json:"agent-version"`
	Go        string `json:"go-version"`
}

// Get returns the Version object with the relevant information
func Get() Version {
	return Version{
		Operator:  version,
		BuildDate: buildDate,
		Jaeger:    DefaultJaeger(),
		Agent:     DefaultAgent(),
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

// DefaultJaeger returns the default Jaeger to use when no versions are specified via CLI or configuration
func DefaultJaeger() string {
	if len(defaultJaeger) > 0 {
		// this should always be set, as it's specified during the build
		return defaultJaeger
	}

	// fallback value, useful for tests
	return "0.0.0"
}

// DefaultAgent returns the default Jaeger to use when no versions are specified via CLI or configuration
func DefaultAgent() string {
	if len(defaultAgent) > 0 {
		// this should always be set, as it's specified during the build
		return defaultAgent
	}

	// fallback value, useful for tests
	return "0.0.0"
}

// DefaultJaegerMajorMinor returns the major.minor format of the default Jaeger version
func DefaultJaegerMajorMinor() string {
	version := DefaultJaeger()
	return version[:strings.LastIndex(version, ".")]
}
