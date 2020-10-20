package otelconfig

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	configFileName     = "config.yaml"
	configFlagName     = "--config="
	configFileLocation = "/etc/jaeger/otel/"
	configFlagWithFile = configFlagName + configFileLocation + configFileName
	configMapKey       = "config"
)

// ShouldCreate returns true if the OTEL config should be created.
func ShouldCreate(jaeger *v1.Jaeger, opts v1.Options, otelCfg map[string]interface{}) bool {
	if _, exists := opts.Map()["config"]; exists {
		jaeger.Logger().Info("OpenTelemetry config will not be created. The config is explicitly provided in the options.")
		return false
	}
	return len(otelCfg) > 0
}

// Get returns a OTEL config maps for a Jaeger instance.
func Get(jaeger *v1.Jaeger) []corev1.ConfigMap {
	var cms []corev1.ConfigMap
	c := createIfNeeded(jaeger, "agent", jaeger.Spec.Agent.Options, jaeger.Spec.Agent.Config)
	if c != nil {
		cms = append(cms, *c)
	}
	c = createIfNeeded(jaeger, "collector", jaeger.Spec.Collector.Options, jaeger.Spec.Collector.Config)
	if c != nil {
		cms = append(cms, *c)
	}
	c = createIfNeeded(jaeger, "ingester", jaeger.Spec.Ingester.Options, jaeger.Spec.Ingester.Config)
	if c != nil {
		cms = append(cms, *c)
	}
	c = createIfNeeded(jaeger, "all-in-one", jaeger.Spec.AllInOne.Options, jaeger.Spec.AllInOne.Config)
	if c != nil {
		cms = append(cms, *c)
	}
	return cms
}

func getMap(log *logrus.Entry, otelConfig v1.FreeForm) (map[string]interface{}, error) {
	m, err := otelConfig.GetMap()
	if err != nil {
		log.WithField("error", err).
			Errorf("Could not parse OTEL config, config map will not be created")
	}
	return m, err
}

func createIfNeeded(jaeger *v1.Jaeger, component string, opts v1.Options, otelConfig v1.FreeForm) *corev1.ConfigMap {
	m, err := getMap(jaeger.Logger().WithField("component", component), otelConfig)
	if err != nil {
		return nil
	}
	if ShouldCreate(jaeger, opts, m) {
		c, err := create(jaeger, component, m)
		if err != nil {
			return nil
		}
		return c
	}
	return nil
}

func create(jaeger *v1.Jaeger, component string, otelConfig map[string]interface{}) (*corev1.ConfigMap, error) {
	cfgYml, err := yaml.Marshal(otelConfig)
	if err != nil {
		jaeger.Logger().
			WithField("component", component).
			WithField("config", otelConfig).
			WithField("err", err).
			Errorf("Could not marshall collector config to yaml")
		return nil, err
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-otel-config", jaeger.Name, component),
			Namespace: jaeger.Namespace,
			Labels:    util.Labels(fmt.Sprintf("%s-%s-otel-config", jaeger.Name, component), fmt.Sprintf("%s-otel-config", component), *jaeger),
			OwnerReferences: []metav1.OwnerReference{
				util.AsOwner(jaeger),
			},
		},
		Data: map[string]string{
			configMapKey: string(cfgYml),
		},
	}, nil
}

func upsert(jaeger *v1.Jaeger, component string, commonSpec *v1.JaegerCommonSpec, args *[]string) {
	volumes := []corev1.Volume{{
		Name: volumeName(jaeger, component),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("%s-%s-otel-config", jaeger.Name, component),
				},
				Items: []corev1.KeyToPath{
					{
						Key:  configMapKey,
						Path: configFileName,
					},
				},
			},
		},
	}}
	volumeMounts := []corev1.VolumeMount{{
		Name:      volumeName(jaeger, component),
		MountPath: configFileLocation,
		ReadOnly:  true,
	}}

	// remove stale volumes, keeping only the one we assembled here
	commonSpec.Volumes = util.RemoveDuplicatedVolumes(append(volumes, commonSpec.Volumes...))
	commonSpec.VolumeMounts = util.RemoveDuplicatedVolumeMounts(append(volumeMounts, commonSpec.VolumeMounts...))

	*args = append(*args, configFlagWithFile)
}

func remove(jaeger *v1.Jaeger, component string, commonSpec *v1.JaegerCommonSpec, args *[]string) {
	name := volumeName(jaeger, component)

	volumes := []corev1.Volume{}
	for _, volume := range commonSpec.Volumes {
		if volume.Name != name {
			volumes = append(volumes, volume)
		}
	}
	commonSpec.Volumes = volumes

	mounts := []corev1.VolumeMount{}
	for _, mount := range commonSpec.VolumeMounts {
		if mount.Name != name {
			mounts = append(mounts, mount)
		}
	}
	commonSpec.VolumeMounts = mounts

	if args != nil {
		newArgs := []string{}
		for _, arg := range *args {
			if !strings.Contains(arg, configFileLocation) {
				newArgs = append(newArgs, arg)
			}
		}
		*args = newArgs
	}
}

// Sync creates, updates or deletes spec and args entries for the component based on the given instance, opts and configuration.
func Sync(jaeger *v1.Jaeger, component string, opts v1.Options, cfg map[string]interface{}, spec *v1.JaegerCommonSpec, args *[]string) {
	if ShouldCreate(jaeger, opts, cfg) {
		upsert(jaeger, component, spec, args)
	} else {
		remove(jaeger, component, spec, args)
	}
}

func volumeName(jaeger *v1.Jaeger, component string) string {
	return util.DNSName(util.Truncate("%s-%s-otel-config", 63, jaeger.Name, component))
}
