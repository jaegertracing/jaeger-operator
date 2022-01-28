package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/jaegertracing/jaeger-operator/tests/assert-jobs/utils/logger"
)

const (
	flagcronJobName   = "cronjob"
	flagVerbose       = "verbose"
	flagNamespace     = "namespace"
	flagKubeconfig    = "kubeconfig"
	flagRetryInterval = "retry-interval"
	flagTimeout       = "timeout"
)

var log logrus.Logger

// Check if a CronJob exists in the given Kubernetes context
// clientset: Kubernetes API client
func checkCronJobExists(clientset *kubernetes.Clientset) error {
	cronjobName := viper.GetString(flagcronJobName)
	namespace := viper.GetString(flagNamespace)
	retryInterval := viper.GetDuration(flagRetryInterval)
	timeout := viper.GetDuration(flagTimeout)

	logrus.Debugln("Checking if the", cronjobName, "CronJob exists")

	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		cronjobs, err := clientset.BatchV1beta1().CronJobs(namespace).List(ctxWithTimeout, metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				logrus.Debug("No cronjobs were found")
				return false, nil
			}
			return false, nil
		}

		for _, cronjob := range cronjobs.Items {
			if cronjob.Name == cronjobName {
				return true, nil
			}
		}

		logrus.Warningln("The Cronjob", cronjobName, "was not found")
		logrus.Debugln("Found cronjobs:")
		for _, cronjob := range cronjobs.Items {
			logrus.Debugln("\t", cronjob.Name)
		}
		return false, nil
	})

	logrus.Infoln("Cronjob", cronjobName, "found successfully")
	return err
}

// Wait for the next job from the given CronJob
// clientset: Kubernetes API client
func waitForNextJob(clientset *kubernetes.Clientset) error {
	cronjobName := viper.GetString(flagcronJobName)
	namespace := viper.GetString(flagNamespace)
	retryInterval := viper.GetDuration(flagRetryInterval)
	timeout := viper.GetDuration(flagTimeout)
	start := time.Now()

	logrus.Debugln("Waiting for the next scheduled job from", cronjobName, "cronjob")
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		jobList, err := clientset.BatchV1().Jobs(namespace).List(ctxWithTimeout, metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				logrus.Debug("No jobs provided by the Kubernetes API")
				return false, nil
			}
			return false, err
		}

		for _, j := range jobList.Items {
			for _, r := range j.OwnerReferences {
				// Check if this job is related to the desired CronJob
				if cronjobName != r.Name {
					continue
				}

				// Check if the job has finished properly
				if j.Status.Succeeded == 0 || j.Status.Failed != 0 || j.Status.Active != 0 {
					continue
				}

				timeSinceCompleted := j.Status.CompletionTime.Sub(start)

				// The job finished before this program started. We are interested in a newer execution
				if timeSinceCompleted <= 0 {
					continue
				}

				return true, nil

			}
		}

		logrus.Debugln("Waiting for next job from", cronjobName, "to succeed")
		return false, nil
	})
	logrus.Infoln("Job of owner", cronjobName, "succeeded after", cronjobName, time.Since(start))
	return err
}

/// Get the Kubernetes client from the environment configuration
func getKubernetesClient() *kubernetes.Clientset {
	// Use the current context
	config, err := clientcmd.BuildConfigFromFlags("", viper.GetString(flagKubeconfig))
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

// Init the CMD and return error if something didn't go properly
func initCmd() error {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault(flagcronJobName, "")
	flag.String(flagcronJobName, "", "Cronjob name")

	viper.SetDefault(flagRetryInterval, time.Second*10)
	flag.Duration(flagRetryInterval, time.Second*10, "Retry interval")

	viper.SetDefault(flagTimeout, time.Hour)
	flag.Duration(flagTimeout, time.Hour, "Timeout")

	viper.SetDefault(flagNamespace, "default")
	flag.String(flagNamespace, "", "Kubernetes namespace")

	viper.SetDefault(flagVerbose, false)
	flag.Bool(flagVerbose, false, "Enable verbosity")

	viper.SetDefault(flagKubeconfig, filepath.Join(homedir.HomeDir(), ".kube", "config"))
	flag.String("kubeconfig", "", "absolute path to the kubeconfig file")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return err
	}

	if viper.GetString(flagcronJobName) == "" {
		return fmt.Errorf(fmt.Sprintf("Parameter --%s must be set", flagcronJobName))
	}

	if _, err := os.Stat(viper.GetString(flagKubeconfig)); err != nil {
		return fmt.Errorf(fmt.Sprintf("%s file does not exists. Point to the correct one using the --%s flag", viper.GetString(flagKubeconfig), flagKubeconfig))
	}

	return nil
}

func main() {
	err := initCmd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log = *logger.InitLog(viper.GetBool(flagVerbose))
	clientset := getKubernetesClient()

	err = checkCronJobExists(clientset)
	if err != nil {
		log.Errorln(err)
		os.Exit(1)
	}

	err = waitForNextJob(clientset)
	if err != nil {
		log.Fatal(err)
	}
}
