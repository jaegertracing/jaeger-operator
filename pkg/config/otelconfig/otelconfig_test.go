package otelconfig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestShouldCreate(t *testing.T) {
	tests := []struct {
		opts     v1.Options
		otelCfg  v1.FreeForm
		expected bool
	}{
		{
			expected: false,
			opts:     v1.NewOptions(map[string]interface{}{"config": "/etc/config.yaml"}),
			otelCfg:  v1.NewFreeForm(map[string]interface{}{"foo": "bar"}),
		},
		{
			expected: true,
			otelCfg:  v1.NewFreeForm(map[string]interface{}{"foo": "bar"}),
		},
		{
			expected: true,
			opts:     v1.NewOptions(map[string]interface{}{"someflag": "val"}),
			otelCfg:  v1.NewFreeForm(map[string]interface{}{"foo": "bar"}),
		},
		{
			expected: false,
			opts:     v1.NewOptions(map[string]interface{}{}),
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			m, err := test.otelCfg.GetMap()
			require.NoError(t, err)
			shouldCreate := ShouldCreate(v1.NewJaeger(types.NamespacedName{}), test.opts, m)
			assert.Equal(t, test.expected, shouldCreate)
		})
	}
}

func TestGet(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "jaeger"})
	j.Spec.Agent.Config = v1.NewFreeForm(map[string]interface{}{"processors": "bar"})
	j.Spec.Collector.Config = v1.NewFreeForm(map[string]interface{}{"exporters": "bar"})
	j.Spec.Ingester.Config = v1.NewFreeForm(map[string]interface{}{"receivers": "bar"})
	cms := Get(j)
	assert.Equal(t, 3, len(cms))
	assert.Equal(t, "jaeger-agent-otel-config", cms[0].Name)
	assert.Equal(t, "jaeger-collector-otel-config", cms[1].Name)
	assert.Equal(t, "jaeger-ingester-otel-config", cms[2].Name)

	m, err := j.Spec.Agent.Config.GetMap()
	require.NoError(t, err)
	yamlCfg, err := yaml.Marshal(m)
	require.NoError(t, err)
	assert.Equal(t, string(yamlCfg), cms[0].Data["config"])
	m, err = j.Spec.Collector.Config.GetMap()
	require.NoError(t, err)
	yamlCfg, err = yaml.Marshal(m)
	require.NoError(t, err)
	assert.Equal(t, string(yamlCfg), cms[1].Data["config"])
	m, err = j.Spec.Ingester.Config.GetMap()
	require.NoError(t, err)
	yamlCfg, err = yaml.Marshal(m)
	require.NoError(t, err)
	assert.Equal(t, string(yamlCfg), cms[2].Data["config"])
}

func TestUpdate(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "jaeger"})
	args := []string{}
	commonSpec := &v1.JaegerCommonSpec{}
	Update(j, "agent", commonSpec, &args)
	assert.Equal(t, []string{configFlagWithFile}, args)
	assert.Equal(t, "jaeger-agent-otel-config", commonSpec.Volumes[0].Name)
	assert.Equal(t, "jaeger-agent-otel-config", commonSpec.VolumeMounts[0].Name)
	assert.Equal(t, []corev1.KeyToPath{{Key: "config", Path: configFileName}}, commonSpec.Volumes[0].ConfigMap.Items)
	assert.Equal(t, configFileLocation, commonSpec.VolumeMounts[0].MountPath)
}
