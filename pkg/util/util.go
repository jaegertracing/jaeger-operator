package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/version"
)

// RemoveDuplicatedVolumes returns a unique list of Volumes based on Volume names. Only the first item is kept.
func RemoveDuplicatedVolumes(volumes []corev1.Volume) []corev1.Volume {
	var results []corev1.Volume
	existing := map[string]bool{}

	for _, volume := range volumes {
		if existing[volume.Name] {
			continue
		}
		results = append(results, volume)
		existing[volume.Name] = true
	}
	// Return the new slice.
	return results
}

// removeDuplicatedVolumeMounts returns a unique list based on the item names. Only the first item is kept.
func removeDuplicatedVolumeMounts(volumeMounts []corev1.VolumeMount) []corev1.VolumeMount {
	var results []corev1.VolumeMount
	existing := map[string]bool{}

	for _, volumeMount := range volumeMounts {
		if existing[volumeMount.Name] {
			continue
		}
		results = append(results, volumeMount)
		existing[volumeMount.Name] = true
	}
	// Return the new slice.
	return results
}

// Merge returns a merged version of the list of JaegerCommonSpec instances with most specific first
func Merge(commonSpecs []v1.JaegerCommonSpec) *v1.JaegerCommonSpec {
	annotations := make(map[string]string)
	labels := make(map[string]string)
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	resources := &corev1.ResourceRequirements{}
	var affinity *corev1.Affinity
	var tolerations []corev1.Toleration
	var securityContext *corev1.PodSecurityContext
	var serviceAccount string

	for _, commonSpec := range commonSpecs {
		// Merge annotations
		for k, v := range commonSpec.Annotations {
			// Only use the value if key has not already been used
			if _, ok := annotations[k]; !ok {
				annotations[k] = v
			}
		}
		// Merge labels
		for k, v := range commonSpec.Labels {
			// Only use the value if key has not already been used
			if _, ok := labels[k]; !ok {
				labels[k] = v
			}
		}
		volumeMounts = append(volumeMounts, commonSpec.VolumeMounts...)
		volumes = append(volumes, commonSpec.Volumes...)

		// Merge resources
		MergeResources(resources, commonSpec.Resources)

		// Set the affinity based on the most specific definition available
		if affinity == nil {
			affinity = commonSpec.Affinity
		}

		tolerations = append(tolerations, commonSpec.Tolerations...)

		if securityContext == nil {
			securityContext = commonSpec.SecurityContext
		}

		if serviceAccount == "" {
			serviceAccount = commonSpec.ServiceAccount
		}
	}

	return &v1.JaegerCommonSpec{
		Annotations:     annotations,
		Labels:          labels,
		VolumeMounts:    removeDuplicatedVolumeMounts(volumeMounts),
		Volumes:         RemoveDuplicatedVolumes(volumes),
		Resources:       *resources,
		Affinity:        affinity,
		Tolerations:     tolerations,
		SecurityContext: securityContext,
		ServiceAccount:  serviceAccount,
	}
}

// MergeResources returns a merged version of two resource requirements
func MergeResources(resources *corev1.ResourceRequirements, res corev1.ResourceRequirements) {

	for k, v := range res.Limits {
		if _, ok := resources.Limits[k]; !ok {
			if resources.Limits == nil {
				resources.Limits = make(corev1.ResourceList)
			}
			resources.Limits[k] = v
		}
	}

	for k, v := range res.Requests {
		if _, ok := resources.Requests[k]; !ok {
			if resources.Requests == nil {
				resources.Requests = make(corev1.ResourceList)
			}
			resources.Requests[k] = v
		}
	}
}

// AsOwner returns owner reference for jaeger
func AsOwner(jaeger *v1.Jaeger) metav1.OwnerReference {
	b := true
	return metav1.OwnerReference{
		APIVersion: jaeger.APIVersion,
		Kind:       jaeger.Kind,
		Name:       jaeger.Name,
		UID:        jaeger.UID,
		Controller: &b,
	}
}

// Labels returns recommended labels
func Labels(name, component string, jaeger v1.Jaeger) map[string]string {
	return map[string]string{
		"app":                          "jaeger", // kept for backwards compatibility, remove by version 2.0
		"app.kubernetes.io/name":       Truncate(name, 63),
		"app.kubernetes.io/instance":   Truncate(jaeger.Name, 63),
		"app.kubernetes.io/component":  Truncate(component, 63),
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",

		// the 'version' label is out for now for two reasons:
		// 1. https://github.com/jaegertracing/jaeger-operator/issues/166
		// 2. these labels are also used as selectors, and as such, need to be consistent... this
		// might be a problem once we support updating the jaeger version
	}
}

// GetEsHostname return first ES hostname from options map
func GetEsHostname(opts map[string]string) string {
	urls, ok := opts["es.server-urls"]
	if !ok {
		return ""
	}
	urlArr := strings.Split(urls, ",")
	return urlArr[0]
}

// FindItem returns the first item matching the given prefix
func FindItem(prefix string, args []string) string {
	for _, v := range args {
		if strings.HasPrefix(v, prefix) {
			return v
		}
	}

	return ""
}

// ReplaceArgument replace argument value with given value.
func ReplaceArgument(prefix string, newValue string, args []string) int {
	found := 0
	for argIndex, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			args[argIndex] = newValue
			found++
		}
	}
	return found
}

// GetPort returns a port, either from supplied default port, or extracted from supplied arg value
func GetPort(arg string, args []string, port int32) int32 {
	portArg := FindItem(arg, args)
	if len(portArg) > 0 {
		i := strings.Index(portArg, ":")
		if i > -1 {
			newPort, err := strconv.ParseInt(portArg[i+1:], 10, 32)
			if err == nil {
				port = int32(newPort)
			}
		}
	}

	return port
}

// InitObjectMeta will set the required default settings to
// kubernetes objects metadata if is required.
func InitObjectMeta(obj metav1.Object) {
	if obj.GetLabels() == nil {
		obj.SetLabels(map[string]string{})
	}

	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(map[string]string{})
	}
}

// ImageName returns the image associated with the supplied image if defined, otherwise
// uses the parameter name to retrieve the value. If the parameter value does not
// include a tag/digest, the Jaeger version will be appended.
func ImageName(image, param string) string {
	if image == "" {
		param := viper.GetString(param)
		if strings.IndexByte(param, ':') == -1 {
			image = fmt.Sprintf("%s:%s", param, version.Get().Jaeger)
		} else {
			image = param
		}
	}
	return image
}

// RemoveEmptyVars removes empty variables from the input slice.
func RemoveEmptyVars(envVars []corev1.EnvVar) []corev1.EnvVar {
	var notEmpty []corev1.EnvVar
	for _, v := range envVars {
		if v.Value != "" || v.ValueFrom != nil {
			notEmpty = append(notEmpty, v)
		}
	}
	return notEmpty
}

// CreateEnvsFromSecret adds env from secret name.
func CreateEnvsFromSecret(secretName string) []corev1.EnvFromSource {
	var envs []corev1.EnvFromSource
	if len(secretName) > 0 {
		envs = append(envs, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
			},
		})
	}
	return envs
}

// GenerateProxySecret generate random secret key for oauth proxy cookie.
func GenerateProxySecret() (string, error) {
	const secretLength = 16
	randString := make([]byte, secretLength)
	_, err := rand.Read(randString)
	if err != nil {
		// If we cannot generate random, return fixed.
		return "", err
	}
	base64Secret := base64.StdEncoding.EncodeToString(randString)
	return base64Secret, nil

}
