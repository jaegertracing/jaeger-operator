package generate

import (
	"context"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_json "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/cmd/start"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
)

// NewGenerateCommand starts the Jaeger Operator
func NewGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "(experimental) Generate YAML manifests from Jaeger CRD",
		Long: `Generate YAML manifests from Jaeger CRD.

Defaults to reading Jaeger CRD from standard input and writing the manifest file to standard output, override with --cr <filename> and --output <filename>.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: generate,
	}

	start.AddFlags(cmd)
	cmd.Flags().String("cr", "/dev/stdin", "Input Jaeger CRD")
	cmd.Flags().String("output", "/dev/stdout", "Where to print the generated YAML documents")

	return cmd
}

func createSpecFromYAML(filename string) (*v1.Jaeger, error) {
	// #nosec   G304: Potential file inclusion via variable
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var spec v1.Jaeger
	decoder := yaml.NewYAMLOrJSONDecoder(f, 8192)
	if err := decoder.Decode(&spec); err != nil && err != io.EOF {
		return nil, err
	}

	return &spec, nil
}

func generate(_ *cobra.Command, _ []string) error {
	level, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}

	input := viper.GetString("cr")
	if input == "/dev/stdin" {
		// Reading from stdin by default is neat when running as a
		// container instead of a binary, but is confusing when no
		// input is sent and the program just hangs
		log.Info("Reading Jaeger CRD from standard input (use --cr <filename> to override)")
	}

	spec, err := createSpecFromYAML(input)
	if err != nil {
		return err
	}

	s := strategy.For(context.Background(), spec)

	outputName := viper.GetString("output")
	out, err := os.OpenFile(outputName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer out.Close()

	encoder := k8s_json.NewYAMLSerializer(k8s_json.DefaultMetaFactory, nil, nil)
	for _, obj := range s.All() {
		// OwnerReferences normally references the CR, but it is not a
		// resource in the cluster so we must remove it

		type f interface {
			SetOwnerReferences(references []metav1.OwnerReference)
		}

		meta := obj.(f)
		meta.SetOwnerReferences(nil)

		fmt.Fprintln(out, "---")
		if err := encoder.Encode(obj, out); err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
