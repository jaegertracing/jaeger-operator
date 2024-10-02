package inject

import (
	"fmt"
	"sort"
	"strings"

	"github.com/operator-framework/operator-lib/proxy"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/autodetect"
	"github.com/jaegertracing/jaeger-operator/pkg/config/ca"
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

func proxyInitArguments(jaeger *v1.Jaeger) []string {
	secret := util.GenerateProxySecret()
	args := []string{
		fmt.Sprintf("--cookie-secret=%s", secret),
		"--https-address=:8443",
		fmt.Sprintf("--openshift-service-account=%s", account.OAuthProxyAccountNameFor(jaeger)),
		"--provider=openshift",
		"--tls-cert=/etc/tls/private/tls.crt",
		"--tls-key=/etc/tls/private/tls.key",
		"--upstream=http://localhost:16686",
	}
	if jaeger.Spec.Ingress.Openshift.Timeout != nil {
		args = append(args, fmt.Sprintf("--upstream-timeout=%s", (*jaeger.Spec.Ingress.Openshift.Timeout).Duration.String()))
	}
	return args
}

func getOAuthProxyContainer(jaeger *v1.Jaeger) corev1.Container {
	commonSpec := util.Merge([]v1.JaegerCommonSpec{jaeger.Spec.Ingress.JaegerCommonSpec, jaeger.Spec.JaegerCommonSpec})
	ca.Update(jaeger, commonSpec)

	args := proxyInitArguments(jaeger)
	volumeMounts := []corev1.VolumeMount{{
		MountPath: "/etc/tls/private",
		Name:      service.GetTLSSecretNameForQueryService(jaeger),
	}}

	// if we have the trusted-ca volume, we mount it in the oauth proxy as well
	trustedCAVolumeName := ca.TrustedCAName(jaeger)
	for _, v := range commonSpec.VolumeMounts {
		if v.Name == trustedCAVolumeName {
			jaeger.Logger().V(-1).Info("found a volume mount with the trusted-ca")
			volumeMounts = append(volumeMounts, v)
		}
	}

	if len(jaeger.Spec.Ingress.Openshift.HtpasswdFile) > 0 {
		args = append(args, fmt.Sprintf("--htpasswd-file=%s", jaeger.Spec.Ingress.Openshift.HtpasswdFile))
		args = append(args, "--display-htpasswd-form=false")

		// we can only get VolumeMounts from the top-level node
		volumeMounts = append(volumeMounts, jaeger.Spec.JaegerCommonSpec.VolumeMounts...)
	}

	if jaeger.Spec.Ingress.Openshift.SAR != nil && len(strings.TrimSpace(*jaeger.Spec.Ingress.Openshift.SAR)) > 0 {
		args = append(args, fmt.Sprintf("--openshift-sar=%s", *jaeger.Spec.Ingress.Openshift.SAR))
	}

	if len(jaeger.Spec.Ingress.Openshift.DelegateUrls) > 0 && autodetect.OperatorConfiguration.IsAuthDelegatorAvailable() {
		args = append(args, fmt.Sprintf("--openshift-delegate-urls=%s", jaeger.Spec.Ingress.Openshift.DelegateUrls))
	}

	args = append(args, jaeger.Spec.Ingress.Options.ToArgs()...)

	sort.Strings(args)

	return corev1.Container{
		Image:        viper.GetString(v1.FlagOpenShiftOauthProxyImage),
		Name:         "oauth-proxy",
		Args:         args,
		VolumeMounts: volumeMounts,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 8443,
				Name:          "public",
			},
		},
		Resources: commonSpec.Resources,
		Env:       proxy.ReadProxyVarsFromEnv(),
	}
}

// PropagateOAuthCookieSecret preserve the generated oauth cookie across multiple reconciliations
func PropagateOAuthCookieSecret(specSrc, specDst appsv1.DeploymentSpec) appsv1.DeploymentSpec {
	spec := specDst.DeepCopy()
	secretArg := ""
	// Find secretArg from old object
	for _, container := range specSrc.Template.Spec.Containers {
		if container.Name == "oauth-proxy" {
			secretArg = util.FindItem("--cookie-secret=", container.Args)
			break
		}
	}
	// Found the cookie secretArg parameter, replace argument.
	if secretArg != "" {
		for i, container := range spec.Template.Spec.Containers {
			if container.Name == "oauth-proxy" {
				util.ReplaceArgument("--cookie-secret", secretArg, spec.Template.Spec.Containers[i].Args)
			}
		}
	}
	return *spec
}
