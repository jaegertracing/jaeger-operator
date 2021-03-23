package utils

import (
	"time"

	"github.com/spf13/viper"
)

const (
	timeoutKey       = "timeout"
	retryIntervalKey = "retry-interval"
)

const (
	timeOutDefault       = 5
	retryIntervalDefault = 120
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
	viper.AutomaticEnv()
	viper.SetDefault(timeoutKey, retryIntervalDefault)
	viper.SetDefault(retryIntervalKey, timeOutDefault)
	params.RetryInterval = time.Duration(viper.GetInt(retryIntervalKey)) * time.Second
	params.Timeout = time.Duration(viper.GetInt(timeoutKey)) * time.Second
}
