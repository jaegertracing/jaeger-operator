package utils

import (
	"time"

	"github.com/spf13/viper"
)

const (
	envTimeoutKey    = "TIMEOUT"
	envRetryInterval = "RETRY_INTERVAL"
)

const (
	timeOutDefault       = 60
	retryIntervalDefault = 8
)

//TestParams contains all general parameters of the test job
type TestParams struct {
	Timeout       time.Duration
	RetryInterval time.Duration
}

//NewParameters create a new TestParams structure
func NewParameters() *TestParams {
	return &TestParams{}
}

//Parse the environment variables and fill the structure with the parameters
func (params *TestParams) Parse() {
	viper.SetDefault(envTimeoutKey, timeOutDefault)
	viper.SetDefault(envRetryInterval, retryIntervalDefault)
	params.RetryInterval = time.Duration(viper.GetInt(envRetryInterval)) * time.Second
	params.Timeout = time.Duration(viper.GetInt(envTimeoutKey)) * time.Second
}
