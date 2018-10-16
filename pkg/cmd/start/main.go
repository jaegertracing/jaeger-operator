package start

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	stub "github.com/jaegertracing/jaeger-operator/pkg/stub"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewStartCommand starts the Jaeger Operator
func NewStartCommand() *cobra.Command {

	const (

		// command
		command          = "start"
		shortDescription = "Starts a new Jaeger Operator"
		longDescription

		// flags
		shortHand = ""
		// images
		agentImage     = "jaeger-agent-image"
		queryImage     = "jaeger-query-image"
		collectorImage = "jaeger-collector-image"
		allInOneImage  = "jaeger-all-in-one-image"
		// operators
		jaegerTracing   = "jaegertracing"
		jaegerAgent     = "jaeger-agent"
		jaegerQuery     = "jaeger-query"
		jaegerCollector = "jaeger-collector"
		jaegerAllInOne  = "all-in-one"
		jaegerVersion   = "jaeger-version"
	)

	cmd := &cobra.Command{
		Use:   command,
		Short: shortDescription,
		Long:  longDescription,
		Run: func(cmd *cobra.Command, args []string) {
			start(cmd, args)
		},
	}

	cmd.Flags().StringP(jaegerVersion, shortHand, version.DefaultJaeger(), "The Jaeger version to use")
	viper.BindPFlag(jaegerVersion, cmd.Flags().Lookup(jaegerVersion))

	cmd.Flags().StringP(agentImage, shortHand, jaegerTracing+"/"+jaegerAgent, "The Docker image for the Jaeger Agent")
	viper.BindPFlag(agentImage, cmd.Flags().Lookup(agentImage))

	cmd.Flags().StringP(queryImage, shortHand, jaegerTracing+"/"+jaegerQuery, "The Docker image for the Jaeger Query")
	viper.BindPFlag(queryImage, cmd.Flags().Lookup(queryImage))

	cmd.Flags().StringP(collectorImage, shortHand, jaegerTracing+"/"+jaegerCollector, "The Docker image for the Jaeger Collector")
	viper.BindPFlag(collectorImage, cmd.Flags().Lookup(collectorImage))

	cmd.Flags().StringP(allInOneImage, shortHand, jaegerTracing+"/"+jaegerAllInOne, "The Docker image for the Jaeger all-in-one")
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

	resyncPeriod := 5 * time.Second
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}

	apiVersion := fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, v1alpha1.SchemeGroupVersion.Version)
	watch(apiVersion, "Jaeger", namespace, resyncPeriod)
	watch("apps/v1", "Deployment", namespace, resyncPeriod)

	sdk.Handle(stub.NewHandler())
	go sdk.Run(ctx)

	select {
	case <-ch:
		ctx.Done()
		logrus.Info("Jaeger Operator finished")
	}
}

func watch(apiVersion, kind, namespace string, resyncPeriod time.Duration) {
	logrus.Infof("Watching %s, %s, %s, %d", apiVersion, kind, namespace, resyncPeriod)
	sdk.Watch(apiVersion, kind, namespace, resyncPeriod)
}
