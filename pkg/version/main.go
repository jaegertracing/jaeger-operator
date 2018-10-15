package version

import (
	"fmt"
	"runtime"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/viper"
)

const versionKey = "jaeger-version"

var (
	version       string
	buildDate     string
	defaultJaeger string
)

// Version holds this Operator's version as well as the version of some of the components it uses
type Version struct {
	Operator    string `json:"jaeger-operator"`
	BuildDate   string `json:"build-date"`
	Jaeger      string `json:"jaeger-version"`
	Go          string `json:"go-version"`
	OperatorSdk string `json:"operator-sdk-version"`
}

// Get returns the Version object with the relevant information
func Get() Version {
	var jaeger string
	if viper.IsSet(versionKey) {
		jaeger = viper.GetString(versionKey)
	} else {
		jaeger = defaultJaeger
	}

	return Version{
		Operator:    version,
		BuildDate:   buildDate,
		Jaeger:      jaeger,
		Go:          runtime.Version(),
		OperatorSdk: sdkVersion.Version,
	}
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(Operator='%v', BuildDate='%v', Jaeger='%v', Go='%v', OperatorSDK='%v')",
		v.Operator,
		v.BuildDate,
		v.Jaeger,
		v.Go,
		v.OperatorSdk,
	)
}

// DefaultJaeger returns the default Jaeger to use when no versions are specified via CLI or configuration
func DefaultJaeger() string {
	return defaultJaeger
}
