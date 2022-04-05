package storage

import (
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

// UpdateGRPCPlugin configures storage grpc-plugin. It adds storage flags, volume and volume mount for the plugin binary.
func UpdateGRPCPlugin(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec) {
	if jaeger.Spec.Storage.Type != v1.JaegerGRPCPluginStorage {
		return
	}

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

// GetGRPCPluginInitContainers returns init containers for grpc-plugin storage.
func GetGRPCPluginInitContainers(jaeger *v1.Jaeger, commonSpec *v1.JaegerCommonSpec) []corev1.Container {
	if jaeger.Spec.Storage.Type != v1.JaegerGRPCPluginStorage {
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
