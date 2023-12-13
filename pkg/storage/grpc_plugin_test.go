package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

func TestStoragePluginEmptyDirVolume(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.Type = v1.JaegerGRPCPluginStorage
	jaeger.Spec.Storage.Options = v1.NewOptions(map[string]interface{}{"grpc-storage-plugin.binary": "/plugin/test"})

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
	UpdateGRPCPlugin(jaeger, &commonSpec)
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

func TestUpdatesOnlyAffectsGRPCStoragePlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	commonSpec := &v1.JaegerCommonSpec{}
	UpdateGRPCPlugin(jaeger, commonSpec)
	assert.Empty(t, commonSpec.Volumes)
	assert.Empty(t, commonSpec.VolumeMounts)
}

func TestGetInitContainersOnlyAffectsGRPCStoragePlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	assert.Empty(t, GetGRPCPluginInitContainers(jaeger, nil))
}

func TestGetInitContainersGRPCStoragePlugin(t *testing.T) {
	jaeger := v1.NewJaeger(types.NamespacedName{Name: "my-instance"})
	jaeger.Spec.Storage.Type = v1.JaegerGRPCPluginStorage
	jaeger.Spec.Storage.GRPCPlugin.Image = "storage-plugin:1.0"

	commonSpec := v1.JaegerCommonSpec{
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "vol",
				MountPath: "/mnt",
			},
		},
	}
	UpdateGRPCPlugin(jaeger, &commonSpec)
	containers := GetGRPCPluginInitContainers(jaeger, &commonSpec)

	require.Len(t, containers, 1)
	assert.Equal(t, jaeger.Spec.Storage.GRPCPlugin.Image, containers[0].Image, "Init container image must be set as in CR")
	assert.Equal(t, commonSpec.VolumeMounts, containers[0].VolumeMounts, "Init container volume mounts must match common volumes")
}
