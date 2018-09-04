package version

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

// NewVersionCommand creates the command that exposes the version
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Long:  `Print the version and build information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()
			json, err := json.Marshal(info)
			if err != nil {
				return err
			}
			fmt.Println(string(json))

			return nil
		},
	}
}
