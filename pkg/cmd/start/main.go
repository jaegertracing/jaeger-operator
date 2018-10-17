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
		jaegerVersion = "jaeger-jaegerVersion"

		// command
		command          = "start"
		shortDescription = "Starts a new Jaeger Operator"
		longDescription

		// images
		imageNS        = "jaegertracing"
		imageAgent     = "jaeger-agent-image"
		imageQuery     = "jaeger-query-image"
		imageCollector = "jaeger-collector-image"
		imageAllInOne  = "jaeger-all-in-one-image"
	)

	cmd := &cobra.Command{
		Use:   command,
		Short: shortDescription,
		Long:  longDescription,
		Run: func(cmd *cobra.Command, args []string) {
			start(cmd, args)
		},
	}

	cmd.Flags().StringP(jaegerVersion, "", version.DefaultJaeger(), "The Jaeger jaegerVersion to use")
	viper.BindPFlag(jaegerVersion, cmd.Flags().Lookup(jaegerVersion))

	cmd.Flags().StringP(imageAgent, "", imageNS+"/jaeger-agent", "The Docker image for the Jaeger Agent")
	viper.BindPFlag(imageAgent, cmd.Flags().Lookup(imageAgent))

	cmd.Flags().StringP(imageQuery, "", imageNS+"/jaeger-query", "The Docker image for the Jaeger Query")
	viper.BindPFlag(imageQuery, cmd.Flags().Lookup(imageQuery))

	cmd.Flags().StringP(imageCollector, "", imageNS+"/jaeger-collector", "The Docker image for the Jaeger Collector")
	viper.BindPFlag(imageCollector, cmd.Flags().Lookup(imageCollector))

	cmd.Flags().StringP(imageAllInOne, "", imageNS+"/all-in-one", "The Docker image for the Jaeger all-in-one")
	viper.BindPFlag(imageAllInOne, cmd.Flags().Lookup(imageAllInOne))

	return cmd
}

func start(cmd *cobra.Command, args []string) {
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
