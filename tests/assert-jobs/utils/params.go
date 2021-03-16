package utils

import (
	"github.com/spf13/viper"
	"time"
)

const (
	jaegerInstanceNameKey = "JAEGER_INSTANCE_NAME"
	namespaceKey          = "NAMESPACE"
	timeoutKey            = "TIMEOUT"
	retryIntervalKey      = "RETRY_INTERVAL"
)

const (
	timeOutDefault = 5
	retryIntervalDefault = 120
)

//TestParams contains all general parameters of the test job
type TestParams struct {
	JaegerName    string
	Namespace     string
	Timeout       time.Duration
	RetryInterval time.Duration
}

//NewParameters create a new TestParams structure
func NewParameters() *TestParams {
	return &TestParams{}
}

//Parse the environment variables and fill the structure with the parameters
func (params *TestParams) Parse() {
	viper.SetDefault(timeoutKey, retryIntervalDefault)
	viper.SetDefault(retryIntervalKey, timeOutDefault)
	viper.AutomaticEnv()
	params.JaegerName = viper.GetString(jaegerInstanceNameKey)
	params.Namespace = viper.GetString(namespaceKey)
	params.RetryInterval = time.Duration(viper.GetInt(retryIntervalKey))*time.Second
	params.Timeout = time.Duration(viper.GetInt(timeoutKey))*time.Second
}
