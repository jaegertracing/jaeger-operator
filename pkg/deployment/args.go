package deployment

import (
	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func allArgs(optionsList ...v1.Options) []string {
	args := []string{}
	for _, options := range optionsList {
		args = append(args, options.ToArgs()...)
	}
	return args
}
