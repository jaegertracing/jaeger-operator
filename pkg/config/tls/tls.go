package tls

import (
	"fmt"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// Update will mount the tls secret on the collector pod.
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec, options *[]string) {
	if viper.GetString("platform") != v1.FlagPlatformOpenShift {
		return
	}

	volume := corev1.Volume{
		Name: configurationVolumeName(jaeger),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: fmt.Sprintf("%s-tls", service.GetNameForHeadlessCollectorService(jaeger)),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      configurationVolumeName(jaeger),
		MountPath: "/etc/tls-config",
		ReadOnly:  true,
	}
	commonSpec.Volumes = append(commonSpec.Volumes, volume)
	commonSpec.VolumeMounts = append(commonSpec.VolumeMounts, volumeMount)
	*options = append(*options, "--collector.grpc.tls.enabled=true")
	*options = append(*options, "--collector.grpc.tls.cert=/etc/tls-config/tls.crt")
	*options = append(*options, "--collector.grpc.tls.key=/etc/tls-config/tls.key")
}

func configurationVolumeName(jaeger *v1.Jaeger) string {
	return util.DNSName(fmt.Sprintf("%s-collector-tls-config-volume", jaeger.Name))
}
