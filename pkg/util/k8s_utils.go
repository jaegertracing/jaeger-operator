package util

import (
	"fmt"
	"os"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

//
// Migrated from operator-sdk, these are internal in newer versions
//

// ForceRunModeEnv indicates if the operator should be forced to run in either local
// or cluster mode (currently only used for local mode)
var ForceRunModeEnv = "OSDK_FORCE_RUN_MODE"

// RunModeType cluster run mode
type RunModeType string

const (
	// LocalRunMode local run mode
	LocalRunMode RunModeType = "local"
	// ClusterRunMode cluster run mode
	ClusterRunMode RunModeType = "cluster"
)

func isRunModeLocal() bool {
	return os.Getenv(ForceRunModeEnv) == string(LocalRunMode)
}

// ErrRunLocal indicates that the operator is set to run in local mode (this error
// is returned by functions that only work on operators running in cluster mode)
var ErrRunLocal = fmt.Errorf("operator run mode forced to local")

// ErrNoNamespace indicates that a namespace could not be found for the current
// environment
var ErrNoNamespace = fmt.Errorf("namespace not found for current environment")

var log = logf.Log.WithName("k8sutil")

// GetOperatorNamespace returns the namespace the operator should be running in.
func GetOperatorNamespace() (string, error) {
	if isRunModeLocal() {
		return "", ErrRunLocal
	}
	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoNamespace
		}
		return "", err
	}
	ns := strings.TrimSpace(string(nsBytes))
	log.V(1).Info("Found namespace", "Namespace", ns)
	return ns, nil
}
