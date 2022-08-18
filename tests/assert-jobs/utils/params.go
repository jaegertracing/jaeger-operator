package utils

import (
	"time"

	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils/secrets"
)

const (
	envTimeoutKey    = "TIMEOUT"
	envRetryInterval = "RETRY_INTERVAL"
	envSecretPath    = "SECRET_PATH"
)

const (
	timeOutDefault       = 600 // seconds
	retryIntervalDefault = 8   // seconds
	secretDefault        = ""
)

// TestParams contains all general parameters of the test job
type TestParams struct {
	Timeout       time.Duration
	RetryInterval time.Duration
	Secret        string
}

// NewParameters create a new TestParams structure
func NewParameters() *TestParams {
	return &TestParams{}
}

// Parse the environment variables and fill the structure with the parameters
func (params *TestParams) Parse() {
	viper.SetDefault(envTimeoutKey, timeOutDefault)
	viper.SetDefault(envRetryInterval, retryIntervalDefault)
	params.RetryInterval = time.Duration(viper.GetInt(envRetryInterval)) * time.Second
	params.Timeout = time.Duration(viper.GetInt(envTimeoutKey)) * time.Second

	secretPath := viper.GetString(envSecretPath)
	if secretPath == "" {
		params.Secret = secretDefault
	} else {
		params.Secret = secrets.GetToken(secretPath)
	}
}
