package deployment

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func init() {
	viper.SetDefault("jaeger-version", "1.6")
	viper.SetDefault("jaeger-all-in-one-image", "jaegertracing/all-in-one")
}

func TestDefaultAllInOneImage(t *testing.T) {
	viper.Set("jaeger-all-in-one-image", "org/custom-all-in-one-image")
	viper.Set("jaeger-version", "123")
	defer viper.Reset()

	d := NewAllInOne(v1alpha1.NewJaeger("TestAllInOneDefaultImage")).Get()

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "org/custom-all-in-one-image:123", d.Spec.Template.Spec.Containers[0].Image)

	envvars := []v1.EnvVar{
		v1.EnvVar{
			Name:  "SPAN_STORAGE_TYPE",
			Value: "",
		},
		v1.EnvVar{
			Name:  "COLLECTOR_ZIPKIN_HTTP_PORT",
			Value: "9411",
		},
	}
	assert.Equal(t, envvars, d.Spec.Template.Spec.Containers[0].Env)
}

func TestAllInOneAnnotations(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestAllInOneAnnotations")
	jaeger.Spec.Annotations = map[string]string{
		"name":  "operator",
		"hello": "jaeger",
	}
	jaeger.Spec.AllInOne.Annotations = map[string]string{
		"hello":                "world", // Override top level annotation
		"prometheus.io/scrape": "false", // Override implicit value
	}

	allinone := NewAllInOne(jaeger)
	dep := allinone.Get()

	assert.Equal(t, "operator", dep.Spec.Template.Annotations["name"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["sidecar.istio.io/inject"])
	assert.Equal(t, "world", dep.Spec.Template.Annotations["hello"])
	assert.Equal(t, "false", dep.Spec.Template.Annotations["prometheus.io/scrape"])
}

func TestAllInOneHasOwner(t *testing.T) {
	name := "TestAllInOneHasOwner"
	a := NewAllInOne(v1alpha1.NewJaeger(name))
	assert.Equal(t, name, a.Get().ObjectMeta.Name)
}

func TestAllInOneNumberOfServices(t *testing.T) {
	name := "TestNumberOfServices"
	services := NewAllInOne(v1alpha1.NewJaeger(name)).Services()
	assert.Len(t, services, 3) // collector, query, agent

	for _, svc := range services {
		owners := svc.ObjectMeta.OwnerReferences
		assert.Equal(t, name, owners[0].Name)
	}
}

func TestAllInOneVolumeMountsWithVolumes(t *testing.T) {
	name := "TestAllInOneVolumeMountsWithVolumes"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	globalVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name: "globalVolume",
		},
	}

	allInOneVolumes := []v1.Volume{
		v1.Volume{
			Name:         "allInOneVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	allInOneVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name: "allInOneVolume",
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.AllInOne.Volumes = allInOneVolumes
	jaeger.Spec.AllInOne.VolumeMounts = allInOneVolumeMounts
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Volumes, len(append(allInOneVolumes, globalVolumes...)))
	assert.Len(t, podSpec.Containers[0].VolumeMounts, len(append(allInOneVolumeMounts, globalVolumeMounts...)))

	// AllInOne is first while global is second
	assert.Equal(t, "allInOneVolume", podSpec.Volumes[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Volumes[1].Name)
	assert.Equal(t, "allInOneVolume", podSpec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "globalVolume", podSpec.Containers[0].VolumeMounts[1].Name)
}

func TestAllInOneMountGlobalVolumes(t *testing.T) {
	name := "TestAllInOneMountGlobalVolumes"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "globalVolume",
			VolumeSource: v1.VolumeSource{},
		},
	}

	allInOneVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "globalVolume",
			ReadOnly: true,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.AllInOne.VolumeMounts = allInOneVolumeMounts
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// allInOne volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].Name, "globalVolume")
}

func TestAllInOneVolumeMountsWithSameName(t *testing.T) {
	name := "TestAllInOneVolumeMountsWithSameName"

	globalVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "data",
			ReadOnly: true,
		},
	}

	allInOneVolumeMounts := []v1.VolumeMount{
		v1.VolumeMount{
			Name:     "data",
			ReadOnly: false,
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.VolumeMounts = globalVolumeMounts
	jaeger.Spec.AllInOne.VolumeMounts = allInOneVolumeMounts
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Containers[0].VolumeMounts, 1)
	// allInOne volume is mounted
	assert.Equal(t, podSpec.Containers[0].VolumeMounts[0].ReadOnly, false)
}

func TestAllInOneVolumeWithSameName(t *testing.T) {
	name := "TestAllInOneVolumeWithSameName"

	globalVolumes := []v1.Volume{
		v1.Volume{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data1"}},
		},
	}

	allInOneVolumes := []v1.Volume{
		v1.Volume{
			Name:         "data",
			VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/data2"}},
		},
	}

	jaeger := v1alpha1.NewJaeger(name)
	jaeger.Spec.Volumes = globalVolumes
	jaeger.Spec.AllInOne.Volumes = allInOneVolumes
	podSpec := NewAllInOne(jaeger).Get().Spec.Template.Spec

	assert.Len(t, podSpec.Volumes, 1)
	// allInOne volume is mounted
	assert.Equal(t, podSpec.Volumes[0].VolumeSource.HostPath.Path, "/data2")
}
