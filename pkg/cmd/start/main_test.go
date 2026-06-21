package start

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestAddFlagsDefaults(t *testing.T) {
	cmd := &cobra.Command{}
	AddFlags(cmd)
	viper.BindPFlags(cmd.Flags())

	assert.Equal(t, v1.FlagCronJobsVersionBatchV1, viper.GetString(v1.FlagCronJobsVersion))
	assert.Equal(t, v1.FlagAutoscalingVersionV2, viper.GetString(v1.FlagAutoscalingVersion))
}
