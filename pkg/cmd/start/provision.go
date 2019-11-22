package start

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/global"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func provisionOwnJaeger(ctx context.Context, cl client.Client, ns string) {
	tracer := global.TraceProvider().GetTracer(v1.BootstrapTracer)
	ctx, span := tracer.Start(ctx, "provisionOwnJaeger")
	defer span.End()

	// this will provision a simplest instance, with in-memory storage
	// for any other usage, we recommend users to create their own CRs
	j := provisionedCR(types.NamespacedName{Name: "jaeger", Namespace: ns})
	if err := cl.Create(ctx, j); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			span.SetStatus(codes.Internal)
			span.SetAttribute(key.String("error", err.Error()))
			log.WithError(err).Warn("failed to provision the operator's own Jaeger instance")
		}

		span.SetAttribute(key.Bool("provisioned", false))
		return
	}

	span.SetAttribute(key.Bool("provisioned", true))
}

func provisionedCR(nsn types.NamespacedName) *v1.Jaeger {
	return &v1.Jaeger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
			Labels: map[string]string{
				v1.LabelOperatedBy:             viper.GetString(v1.ConfigIdentity),
				"app":                          "jaeger",
				"app.kubernetes.io/name":       nsn.Name,
				"app.kubernetes.io/instance":   nsn.Name,
				"app.kubernetes.io/component":  "service-agent",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
		},
		Spec: v1.JaegerSpec{
			Storage: v1.JaegerStorageSpec{
				Type: "badger",
				Options: v1.NewOptions(map[string]interface{}{
					"badger.ephemeral": "false",
					"directory-key":    "/badger/key",
					"directory-value":  "/badger/data",
				}),
			},
			JaegerCommonSpec: v1.JaegerCommonSpec{
				Volumes: []corev1.Volume{{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "data",
					MountPath: "/badger",
				}},
			},
		},
	}
}
