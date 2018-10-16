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
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a new Jaeger Operator",
		Long:  "Starts a new Jaeger Operator",
		Run: func(cmd *cobra.Command, args []string) {
			start(cmd, args)
		},
	}

	cmd.Flags().StringP("jaeger-version", "", version.DefaultJaeger(), "The Jaeger version to use")
	viper.BindPFlag("jaeger-version", cmd.Flags().Lookup("jaeger-version"))

	cmd.Flags().StringP("jaeger-agent-image", "", "jaegertracing/jaeger-agent", "The Docker image for the Jaeger Agent")
	viper.BindPFlag("jaeger-agent-image", cmd.Flags().Lookup("jaeger-agent-image"))

	cmd.Flags().StringP("jaeger-query-image", "", "jaegertracing/jaeger-query", "The Docker image for the Jaeger Query")
	viper.BindPFlag("jaeger-query-image", cmd.Flags().Lookup("jaeger-query-image"))

	cmd.Flags().StringP("jaeger-collector-image", "", "jaegertracing/jaeger-collector", "The Docker image for the Jaeger Collector")
	viper.BindPFlag("jaeger-collector-image", cmd.Flags().Lookup("jaeger-collector-image"))

	cmd.Flags().StringP("jaeger-all-in-one-image", "", "jaegertracing/all-in-one", "The Docker image for the Jaeger all-in-one")
	viper.BindPFlag("jaeger-all-in-one-image", cmd.Flags().Lookup("jaeger-all-in-one-image"))

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
