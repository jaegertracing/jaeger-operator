package inject

import (
	"fmt"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// OAuthProxy injects an appropriate proxy into the given deployment
func OAuthProxy(jaeger *v1.Jaeger, pod corev1.PodSpec) corev1.PodSpec {
	if jaeger.Spec.Ingress.Security != v1.IngressSecurityOAuthProxy {
		return pod
	}

	pod.ServiceAccountName = account.OAuthProxyAccountNameFor(jaeger)
	pod.Containers = append(pod.Containers, getOAuthProxyContainer(jaeger))
	pod.Volumes = append(pod.Volumes, corev1.Volume{
		Name: service.GetTLSSecretNameForQueryService(jaeger),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: service.GetTLSSecretNameForQueryService(jaeger),
			},
		},
	})
	return pod
}

func getOAuthProxyContainer(jaeger *v1.Jaeger) corev1.Container {
	// keep this sorted!
	// see https://github.com/jaegertracing/jaeger-operator/pull/337
	args := []string{
		"--cookie-secret=SECRET",
		"--https-address=:8443",
		fmt.Sprintf("--openshift-service-account=%s", account.OAuthProxyAccountNameFor(jaeger)),
		"--provider=openshift",
		"--tls-cert=/etc/tls/private/tls.crt",
		"--tls-key=/etc/tls/private/tls.key",
		"--upstream=http://localhost:16686",
	}
	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Ingress.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec})

	return corev1.Container{
		Image: viper.GetString("openshift-oauth-proxy-image"),
		Name:  "oauth-proxy",
		Args:  args,
		VolumeMounts: []corev1.VolumeMount{{
			MountPath: "/etc/tls/private",
			Name:      service.GetTLSSecretNameForQueryService(jaeger),
		}},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 8443,
				Name:          "public",
			},
		},
		Resources: commonSpec.Resources,
	}
}
