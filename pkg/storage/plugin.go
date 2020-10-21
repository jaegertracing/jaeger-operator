package storage

import (
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// Update configures storage
func Update(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec, options *[]string) {
	if jaeger.Spec.Storage.Type != "grpc-plugin" {
		return
	}

	if jaeger.Spec.Storage.GRPCPlugin.ConfigurationFile != "" {
		*options = append(*options, "--grpc-storage-plugin.configuration-file="+jaeger.Spec.Storage.GRPCPlugin.ConfigurationFile)
	}
	*options = append(*options, "--grpc-storage-plugin.binary="+jaeger.Spec.Storage.GRPCPlugin.Binary)

	pluginVolumeName := "plugin-volume"
	volume := corev1.Volume{
		Name: pluginVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      pluginVolumeName,
		MountPath: "/plugin",
	}

	commonSpec.Volumes = append(commonSpec.Volumes, volume)
	commonSpec.VolumeMounts = append(commonSpec.VolumeMounts, volumeMount)
}

// GetInitContainers returns init containers if the storage type requires them. It must be added to all nodes utilizing storage (query, ingester, collector, allInOne)
func GetInitContainers(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec) []corev1.Container {
	if jaeger.Spec.Storage.Type != "grpc-plugin" {
		return nil
	}

	// If the storageType is grpc-plugin, we need a volume to make the
	// binary plugin available to the main container and an init container
	// to install it in the volume
	return []corev1.Container{
		{
			Image:        jaeger.Spec.Storage.GRPCPlugin.Image,
			Name:         "install-plugin",
			VolumeMounts: commonSpec.VolumeMounts,
		},
	}
}
