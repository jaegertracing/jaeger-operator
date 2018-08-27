package deployment

import (
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func allArgs(optionsList ...v1alpha1.Options) []string {
	args := []string{}
	for _, options := range optionsList {
		args = append(args, options.ToArgs()...)
	}
	return args
}
