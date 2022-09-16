package utils

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestGetHTTP polls an endpoint and test the response
func TestGetHTTP(url string, params *TestParams, testFn func(response *http.Response, body []byte) (done bool, err error)) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if params.Secret == "" {
		logrus.Info("No secret provided for the Authorization header")
	} else {
		// This is needed to query the endpoints when using the OAuth Proxy
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", params.Secret))
		logrus.Info("Secret provided for the Authorization header")
	}

	// TODO: https://github.com/jaegertracing/jaeger-operator/issues/951
	client := http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			// #nosec
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	maxRetries := 20
	retries := 0
	failed := false

	logrus.Info("Polling to ", url)

	return wait.Poll(params.RetryInterval, params.Timeout, func() (done bool, err error) {
		logrus.Info("Doing request number ", retries)

		res, err := client.Do(req)
		if err != nil && strings.Contains(err.Error(), "Timeout exceeded") {
			failed = true
			logrus.Warn("Timeout exceeded!")
		} else if res != nil && res.StatusCode != http.StatusOK {
			err = fmt.Errorf("unexpected status code %d", res.StatusCode)
			failed = true
			logrus.Warn("Status code: ", res.StatusCode)
		} else if err != nil {
			failed = true
			logrus.Warn("Something failed during doing the request: ", err.Error())
		}

		if failed {
			failed = false
			retries++
			if retries > maxRetries {
				return false, err
			}
			return false, nil
		}

		body, err := io.ReadAll(res.Body)
		if len(body) == 0 {
			failed = true
			err = fmt.Errorf("empty body response")
			logrus.Warn("Empty body response")
		} else if err != nil {
			failed = true
			logrus.Warn("Something failed reading the response: ", err.Error())
		}

		if failed {
			failed = false
			retries++
			if retries > maxRetries {
				return false, err
			}
			return false, nil
		}

		ok, err := testFn(res, body)
		if ok {
			return true, nil
		}
		retries++

		if retries > maxRetries {
			logrus.Warn("Something failed while executing the test function: ", err.Error())
			return false, err
		}

		if err == nil {
			logrus.Warn("The condition of the test function was not accomplished")
		} else {
			logrus.Warn("There test function returned an error:", err.Error())
		}

		return false, nil
	})
}

// WaitUntilRestAPIAvailable blocks the execution until the Jaeger REST API is
// available (or timeout)
func WaitUntilRestAPIAvailable(jaegerEndpoint string) error {
	logrus.Debugln("Checking the", jaegerEndpoint, "is available")
	transport := &http.Transport{
		// #nosec
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := http.Client{Transport: transport}

	maxRetries := 5
	retries := 0

	err := wait.Poll(time.Second*5, time.Minute*5, func() (done bool, err error) {
		req, err := http.NewRequest(http.MethodGet, jaegerEndpoint, nil)
		if err != nil {
			return false, err
		}

		resp, err := client.Do(req)

		// The GET HTTP verb is not supported by the Jaeger Collector REST API
		// enpoint. An error 404 or 405 means the REST API is there
		if resp != nil && (resp.StatusCode == 404 || resp.StatusCode == 405) {
			logrus.Debugln("Endpoint available!")
			return true, nil
		}

		if err != nil {
			logrus.Warningln("Something failed while reaching", jaegerEndpoint, ":", err)

			if retries < maxRetries {
				retries++
				return false, nil
			}
			return false, err
		}

		logrus.Warningln(jaegerEndpoint, "is not available")
		return false, nil
	})
	return err
}
