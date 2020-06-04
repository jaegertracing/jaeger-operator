package otelconfig

import (
	"fmt"

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

// Update injects required flags and objects to the common spec.
func Update(jaeger *v1.Jaeger, component string, commonSpec *v1.JaegerCommonSpec, args *[]string) {
	volume := corev1.Volume{
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
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName(jaeger, component),
		MountPath: configFileLocation,
		ReadOnly:  true,
	}
	commonSpec.Volumes = append(commonSpec.Volumes, volume)
	commonSpec.VolumeMounts = append(commonSpec.VolumeMounts, volumeMount)
	*args = append(*args, configFlagWithFile)
}

func volumeName(jaeger *v1.Jaeger, component string) string {
	return util.DNSName(util.Truncate("%s-%s-otel-config", 63, jaeger.Name, component))
}
