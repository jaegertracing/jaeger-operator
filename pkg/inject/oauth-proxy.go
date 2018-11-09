package inject

import (
	"fmt"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
)

// OAuthProxy injects an appropriate proxy into the given deployment
func OAuthProxy(jaeger *v1alpha1.Jaeger, dep *appsv1.Deployment) *appsv1.Deployment {
	if jaeger.Spec.Ingress.Security != v1alpha1.IngressSecurityOAuthProxy {
		return dep
	}

	dep.Spec.Template.Spec.ServiceAccountName = account.OAuthProxyAccountNameFor(jaeger)
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, getOAuthProxyContainer(jaeger))
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, v1.Volume{
		Name: service.GetTLSSecretNameForQueryService(jaeger),
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: service.GetTLSSecretNameForQueryService(jaeger),
			},
		},
	})
	return dep
}

func getOAuthProxyContainer(jaeger *v1alpha1.Jaeger) v1.Container {
	return v1.Container{
		Image: viper.GetString("openshift-oauth-proxy-image"),
		Name:  "oauth-proxy",
		Args: []string{
			"--https-address=:8443",
			"--provider=openshift",
			fmt.Sprintf("--openshift-service-account=%s", account.OAuthProxyAccountNameFor(jaeger)),
			"--upstream=http://localhost:16686",
			"--tls-cert=/etc/tls/private/tls.crt",
			"--tls-key=/etc/tls/private/tls.key",
			"--cookie-secret=SECRET",
		},
		VolumeMounts: []v1.VolumeMount{{
			MountPath: "/etc/tls/private",
			Name:      service.GetTLSSecretNameForQueryService(jaeger),
		}},
		Ports: []v1.ContainerPort{
			{
				ContainerPort: 8443,
				Name:          "public",
			},
		},
	}
}
