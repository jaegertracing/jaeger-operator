package inject

import (
	"fmt"
	"sort"

	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/service"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

// OAuthProxy injects an appropriate proxy into the given deployment
func OAuthProxy(jaeger *v1.Jaeger, dep *appsv1.Deployment) *appsv1.Deployment {
	if jaeger.Spec.Ingress.Security != v1.IngressSecurityOAuthProxy {
		return dep
	}

	dep.Spec.Template.Spec.ServiceAccountName = account.OAuthProxyAccountNameFor(jaeger)
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, getOAuthProxyContainer(jaeger))
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: service.GetTLSSecretNameForQueryService(jaeger),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: service.GetTLSSecretNameForQueryService(jaeger),
			},
		},
	})
	return dep
}

func getOAuthProxyContainer(jaeger *v1.Jaeger) corev1.Container {
	args := []string{
		"--cookie-secret=SECRET",
		"--https-address=:8443",
		fmt.Sprintf("--openshift-service-account=%s", account.OAuthProxyAccountNameFor(jaeger)),
		"--provider=openshift",
		"--tls-cert=/etc/tls/private/tls.crt",
		"--tls-key=/etc/tls/private/tls.key",
		"--upstream=http://localhost:16686",
	}

	if len(jaeger.Spec.Ingress.OpenShift.SAR) > 0 {
		args = append(args, fmt.Sprintf("--openshift-sar=%s", jaeger.Spec.Ingress.OpenShift.SAR))
	}

	if len(jaeger.Spec.Ingress.OpenShift.DelegateURLs) > 0 && viper.GetBool("auth-delegator-available") {
		args = append(args, fmt.Sprintf("--openshift-delegate-urls=%s", jaeger.Spec.Ingress.OpenShift.DelegateURLs))
	}

	sort.Strings(args)

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
