package inject

import (
	"github.com/spf13/viper"
)

func setDefaults() {
	viper.SetDefault("jaeger-agent-image", "jaegertracing/jaeger-agent")
}
