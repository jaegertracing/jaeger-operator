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

// KafkaOperatorIntegration holds the if the Kafka Operator integration is enabled.
type KafkaOperatorIntegration int

const (
	// KafkaOperatorIntegrationYes represents the Kafka Operator integration is enabled.
	KafkaOperatorIntegrationYes KafkaOperatorIntegration = iota

	// KafkaOperatorIntegrationNo represents the Kafka Operator integration is disabled.
	KafkaOperatorIntegrationNo
)

func (p KafkaOperatorIntegration) String() string {
	return [...]string{"Yes", "No"}[p]
}

// AuthDelegatorAvailability holds the if the AuthDelegator available.
type AuthDelegatorAvailability int

const (
	// AuthDelegatorAvailabilityYes represents the AuthDelegator is available.
	AuthDelegatorAvailabilityYes AuthDelegatorAvailability = iota

	// AuthDelegatorAvailabilityNo represents the AuthDelegator is not available.
	AuthDelegatorAvailabilityNo

	// AuthDelegatorAvailabilityUnknown represents the AuthDelegator availability is not known.
	AuthDelegatorAvailabilityUnknown
)

func (p AuthDelegatorAvailability) String() string {
	return [...]string{"Yes", "No", "Unknown"}[p]
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

func (c *operatorConfigurationWrapper) SetKafkaIntegration(e interface{}) {
	var integration string
	switch v := e.(type) {
	case string:
		integration = v
	case KafkaOperatorIntegration:
		integration = v.String()
	default:
		integration = KafkaOperatorIntegrationNo.String()
	}

	c.mu.Lock()
	viper.Set(v1.FlagKafkaProvision, integration)
	c.mu.Unlock()
}

func (c *operatorConfigurationWrapper) GetKafkaIntegration() KafkaOperatorIntegration {
	c.mu.RLock()
	e := viper.GetString(v1.FlagKafkaProvision)
	c.mu.RUnlock()

	if strings.ToLower(e) == "yes" {
		return KafkaOperatorIntegrationYes
	}
	return KafkaOperatorIntegrationNo
}

// IsKafkaOperatorIntegrationEnabled returns true if the integration with the
// Kafaka Operator is enabled
func (c *operatorConfigurationWrapper) IsKafkaOperatorIntegrationEnabled() bool {
	return c.GetKafkaIntegration() == KafkaOperatorIntegrationYes
}

func (c *operatorConfigurationWrapper) SetAuthDelegatorAvailability(e interface{}) {
	var availability string
	switch v := e.(type) {
	case string:
		availability = v
	case AuthDelegatorAvailability:
		availability = v.String()
	default:
		availability = AuthDelegatorAvailabilityUnknown.String()
	}

	c.mu.Lock()
	viper.Set(v1.FlagAuthDelegatorAvailability, availability)
	c.mu.Unlock()
}

func (c *operatorConfigurationWrapper) GetAuthDelegator() AuthDelegatorAvailability {
	c.mu.RLock()
	e := viper.GetString(v1.FlagAuthDelegatorAvailability)
	c.mu.RUnlock()

	var available AuthDelegatorAvailability
	switch strings.ToLower(e) {
	case "yes":
		available = AuthDelegatorAvailabilityYes
	case "no":
		available = AuthDelegatorAvailabilityNo
	default:
		available = AuthDelegatorAvailabilityUnknown
	}
	return available
}

// IsAuthDelegatorAvailable returns true if the AuthDelegator is available
func (c *operatorConfigurationWrapper) IsAuthDelegatorAvailable() bool {
	return c.GetAuthDelegator() == AuthDelegatorAvailabilityYes
}

// IsAuthDelegatorAvailable returns true if the AuthDelegator is set
func (c *operatorConfigurationWrapper) IsAuthDelegatorSet() bool {
	return c.GetAuthDelegator() != AuthDelegatorAvailabilityUnknown
}

func (c *operatorConfigurationWrapper) SetOautProxyImage(image string) {
	c.mu.Lock()
	viper.Set(v1.FlagOpenShiftOauthProxyImage, image)
	c.mu.Unlock()
}

func (c *operatorConfigurationWrapper) GetOautProxyImage() string {
	c.mu.RLock()
	image := viper.GetString(v1.FlagOpenShiftOauthProxyImage)
	c.mu.RUnlock()

	return image
}
