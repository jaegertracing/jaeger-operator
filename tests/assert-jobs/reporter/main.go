package main

import (
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

const (
	envOperationName = "OPERATION_NAME"
)

func initTracer() (opentracing.Tracer, io.Closer) {
	cfg, err := config.FromEnv()
	cfg.Reporter.LogSpans = true
	cfg.Sampler = &config.SamplerConfig{
		Type:  "const",
		Param: 1,
	}
	if err != nil {
		panic(err)
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func main() {
	viper.AutomaticEnv()
	operationName := viper.GetString(envOperationName)

	tracer, closer := initTracer()
	defer closer.Close()

	span := tracer.StartSpan(operationName)
	span.Finish()
}
