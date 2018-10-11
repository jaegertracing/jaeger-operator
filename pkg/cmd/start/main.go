package start

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/pkg/stub"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

const (
	shortHand      = ""
	jaegerVersion  = "jaeger-version"
	agentImage     = "jaeger-agent-image"
	queryImage     = "jaeger-query-image"
	collectorImage = "jaeger-collector-image"
	allInOneImage  = "jaeger-all-in-one-image"
)

// NewStartCommand starts the Jaeger Operator
func NewStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a new Jaeger Operator",
		Long:  "Starts a new Jaeger Operator",
		Run: func(cmd *cobra.Command, args []string) {
			start(cmd, args)
		},
	}

	cmd.Flags().StringP(jaegerVersion, shortHand, version.DefaultJaeger(), "The Jaeger version to use")
	viper.BindPFlag(jaegerVersion, cmd.Flags().Lookup(jaegerVersion))

	cmd.Flags().StringP(agentImage, shortHand, "jaegertracing/jaeger-agent", "The Docker image for the Jaeger Agent")
	viper.BindPFlag(agentImage, cmd.Flags().Lookup(agentImage))

	cmd.Flags().StringP(queryImage, shortHand, "jaegertracing/jaeger-query", "The Docker image for the Jaeger Query")
	viper.BindPFlag(queryImage, cmd.Flags().Lookup(queryImage))

	cmd.Flags().StringP(collectorImage, shortHand, "jaegertracing/jaeger-collector", "The Docker image for the Jaeger Collector")
	viper.BindPFlag(collectorImage, cmd.Flags().Lookup(collectorImage))

	cmd.Flags().StringP(allInOneImage, shortHand, "jaegertracing/all-in-one", "The Docker image for the Jaeger all-in-one")
	viper.BindPFlag(allInOneImage, cmd.Flags().Lookup(allInOneImage))

	return cmd
}

func start(cmd *cobra.Command, args []string) {

	const (
		resource = "io.jaegertracing/v1alpha1"
		kind     = "Jaeger"
	)

	var ch = make(chan os.Signal, 0)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	logrus.Infof("Versions used by this operator: %v", version.Get())

	ctx := context.Background()

	sdk.ExposeMetricsPort()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}
	resyncPeriod := 5
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	sdk.Handle(stub.NewHandler())
	go sdk.Run(ctx)

	select {
	case <-ch:
		ctx.Done()
		logrus.Info("Jaeger Operator finished")
	}
}
