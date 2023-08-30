package autodetect

import (
	"strings"
	"sync"

	"github.com/spf13/viper"

	v1 "github.com/jaegertracing/jaeger-operator/apis/v1"
)

// Platform holds the auto-detected running platform.
type Platform int

const (
	// KubernetesPlatform represents the cluster is Kubernetes.
	KubernetesPlatform Platform = iota

	// OpenShiftPlatform represents the cluster is OpenShift.
	OpenShiftPlatform
)

func (p Platform) String() string {
	return [...]string{"Kubernetes", "OpenShift"}[p]
}

// ESOperatorIntegration holds the if the ES Operator integration is enabled.
type ESOperatorIntegration int

const (
	// ESOperatorIntegrationYes represents the ES Operator integration is enabled.
	ESOperatorIntegrationYes ESOperatorIntegration = iota

	// ESOperatorIntegrationNo represents the ES Operator integration is disabled.
	ESOperatorIntegrationNo
)

func (p ESOperatorIntegration) String() string {
	return [...]string{"Yes", "No"}[p]
}

var OperatorConfiguration operatorConfigurationWrapper

type operatorConfigurationWrapper struct {
	mu sync.RWMutex
}

func (c *operatorConfigurationWrapper) SetPlatform(p interface{}) {
	var platform string
	switch v := p.(type) {
	case string:
		platform = v
	case Platform:
		platform = v.String()
	default:
		platform = KubernetesPlatform.String()
	}

	c.mu.Lock()
	viper.Set(v1.FlagPlatform, platform)
	c.mu.Unlock()
}

func (c *operatorConfigurationWrapper) GetPlatform() Platform {
	c.mu.RLock()
	p := viper.GetString(v1.FlagPlatform)
	c.mu.RUnlock()

	if strings.ToLower(p) == "openshift" {
		return OpenShiftPlatform
	}
	return KubernetesPlatform
}

func (c *operatorConfigurationWrapper) IsPlatformAutodetectionEnabled() bool {
	c.mu.RLock()
	p := viper.GetString(v1.FlagPlatform)
	c.mu.RUnlock()

	return strings.EqualFold(p, v1.FlagPlatformAutoDetect)
}

func (c *operatorConfigurationWrapper) SetESIngration(e interface{}) {
	var integration string
	switch v := e.(type) {
	case string:
		integration = v
	case ESOperatorIntegration:
		integration = v.String()
	default:
		integration = ESOperatorIntegrationNo.String()
	}

	c.mu.Lock()
	viper.Set(v1.FlagESProvision, integration)
	c.mu.Unlock()
}

func (c *operatorConfigurationWrapper) GetESPIntegration() ESOperatorIntegration {
	c.mu.RLock()
	e := viper.GetString(v1.FlagESProvision)
	c.mu.RUnlock()

	if strings.ToLower(e) == "yes" {
		return ESOperatorIntegrationYes
	}
	return ESOperatorIntegrationNo
}

// IsESOperatorIntegrationEnabled returns true if the integration with the
// Elasticsearch OpenShift Operator is enabled
func (c *operatorConfigurationWrapper) IsESOperatorIntegrationEnabled() bool {
	return c.GetESPIntegration() == ESOperatorIntegrationYes
}
