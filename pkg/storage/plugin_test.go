package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestStoragePluginEmptyDirVolume(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.Type = "grpc-plugin"
	jaeger.Spec.Storage.GRPCPlugin.Binary = "/plugin/test"

	var options []string
	commonSpec := v1.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "vol",
				MountPath: "/mnt",
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "vol",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}
	Update(jaeger, &commonSpec, &options)
	assert.Len(t, commonSpec.VolumeMounts, 2, "storage.Update for grpc-plugin must add /plugin volume and keep existing mounts")
	assert.Len(t, commonSpec.Volumes, 2, "storage.Update for grpc-plugin must add /plugin volume and keep existing mounts")

	var pluginVolumeName string
	for _, mount := range commonSpec.VolumeMounts {
		if mount.MountPath == "/plugin" {
			pluginVolumeName = mount.Name
		}
	}
	assert.NotEmpty(t, pluginVolumeName, "Did not find a volume mounted at /plugin")

	volume := corev1.Volume{
		Name: pluginVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	assert.Containsf(t, commonSpec.Volumes, volume, "Did not find a volume source for %v", pluginVolumeName)
}

func TestStoragePluginBinary(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.Type = "grpc-plugin"
	jaeger.Spec.Storage.GRPCPlugin.Binary = "/plugin/test"

	var options []string
	var commonSpec v1.JaegerCommonSpec
	Update(jaeger, &commonSpec, &options)
	assert.Len(t, options, 1)
	assert.Equal(t, options[0], "--grpc-storage-plugin.binary=/plugin/test", "Storage plugin binary path must be set when using a gRPC plugin for storage")
}

func TestStoragePluginConfig(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.Type = "grpc-plugin"
	jaeger.Spec.Storage.GRPCPlugin.Binary = "/plugin/test"
	jaeger.Spec.Storage.GRPCPlugin.ConfigurationFile = "/plugin/config.json"

	var commonSpec v1.JaegerCommonSpec
	var options []string
	Update(jaeger, &commonSpec, &options)
	assert.Len(t, options, 2)
	assert.Contains(t, options, "--grpc-storage-plugin.configuration-file=/plugin/config.json", "configuration-file option must be passed to jaeger when defined in CR")
}

// GetInitContainers returns init containers if the storage type requires them. It must be added to all nodes utilizing storage (query, ingester, collector, allInOne)
func TestGetInitContainersOnlyAffectsGRPCStoragePlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	assert.Len(t, GetInitContainers(jaeger, nil), 0)
}

func TestGetInitContainersGRPCStoragePlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.Type = "grpc-plugin"
	jaeger.Spec.Storage.GRPCPlugin.Image = "storage-plugin:1.0"

	commonSpec := v1.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "vol",
				MountPath: "/mnt",
			},
		},
	}
	var options []string
	Update(jaeger, &commonSpec, &options)
	containers := GetInitContainers(jaeger, &commonSpec)

	assert.Len(t, containers, 1)
	assert.Equal(t, jaeger.Spec.Storage.GRPCPlugin.Image, containers[0].Image, "Init container image must be set as in CR")

	assert.Equal(t, commonSpec.VolumeMounts, containers[0].VolumeMounts, "Init container volume mounts must match common volumes")
}
