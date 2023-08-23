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

	p = strings.ToLower(p)

	var platform Platform
	switch p {
	case "openshift":
		platform = OpenShiftPlatform
	default:
		platform = KubernetesPlatform
	}
	return platform
}

func (c *operatorConfigurationWrapper) IsPlatformAutodetectionEnabled() bool {
	c.mu.RLock()
	p := viper.GetString(v1.FlagPlatform)
	c.mu.RUnlock()

	return strings.EqualFold(p, "auto-detect")
}
