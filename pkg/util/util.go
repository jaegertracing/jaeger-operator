package util

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// removeDuplicatedVolumes returns a unique list of Volumes based on Volume names. Only the first item is kept.
func removeDuplicatedVolumes(volumes []corev1.Volume) []corev1.Volume {
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
		mergeResources(resources, commonSpec.Resources)

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
		Volumes:         removeDuplicatedVolumes(volumes),
		Resources:       *resources,
		Affinity:        affinity,
		Tolerations:     tolerations,
		SecurityContext: securityContext,
		ServiceAccount:  serviceAccount,
	}
}

func mergeResources(resources *corev1.ResourceRequirements, res corev1.ResourceRequirements) {

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
		"app":                          "jaeger",
		"app.kubernetes.io/name":       name,
		"app.kubernetes.io/instance":   jaeger.Name,
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/part-of":    "jaeger",
		"app.kubernetes.io/managed-by": "jaeger-operator",
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
