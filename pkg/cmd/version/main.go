package version

import (
	"encoding/json"
	"fmt"
	"runtime"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/cobra"
)

var (
	gitCommit string
	buildDate string
)

// Info holds build information
type Info struct {
	GoVersion          string `json:"go-version"`
	OperatorSdkVersion string `json:"operator-sdk-version"`
	GitCommit          string `json:"commit"`
	BuildDate          string `json:"date"`
}

// Get creates and initialized Info object
func Get() Info {
	return Info{
		GitCommit:          gitCommit,
		BuildDate:          buildDate,
		GoVersion:          runtime.Version(),
		OperatorSdkVersion: sdkVersion.Version,
	}
}

// NewVersionCommand creates the command that exposes the version
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Long:  `Print the version and build information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := Get()
			json, err := json.Marshal(info)
			if err != nil {
				return err
			}
			fmt.Println(string(json))

			return nil
		},
	}
}
