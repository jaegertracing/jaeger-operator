package generate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_json "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/cmd/start"
	"github.com/jaegertracing/jaeger-operator/pkg/strategy"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
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
	defer util.CloseFile(f, &log.Log)

	var spec v1.Jaeger
	decoder := yaml.NewYAMLOrJSONDecoder(f, 8192)
	if err := decoder.Decode(&spec); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return &spec, nil
}

func generate(_ *cobra.Command, _ []string) error {
	var loggingLevel zapcore.Level
	switch strings.ToLower(viper.GetString("log-level")) {
	case "panic":
		loggingLevel = zapcore.PanicLevel
	case "fatal":
		loggingLevel = zapcore.FatalLevel
	case "error":
		loggingLevel = zapcore.ErrorLevel
	case "warn", "warning":
		loggingLevel = zapcore.WarnLevel
	case "info":
		loggingLevel = zapcore.InfoLevel
	case "debug":
		loggingLevel = zapcore.DebugLevel
	}

	opts := zap.Options{
		Development: true,
		Level:       loggingLevel,
	}

	log.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	input := viper.GetString("cr")
	if input == "/dev/stdin" {
		// Reading from stdin by default is neat when running as a
		// container instead of a binary, but is confusing when no
		// input is sent and the program just hangs
		log.Log.Info("Reading Jaeger CRD from standard input (use --cr <filename> to override)")
	}

	spec, err := createSpecFromYAML(input)
	if err != nil {
		return err
	}

	s := strategy.For(context.Background(), spec)

	outputName := viper.GetString("output")
	pathToFile := filepath.Clean(outputName)
	out, err := os.OpenFile(pathToFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	defer util.CloseFile(out, &log.Log)

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
			log.Log.V(3).Info(fmt.Sprintf("Fatal error %s", err))
		}
	}

	return nil
}
