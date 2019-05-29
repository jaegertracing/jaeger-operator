package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	"github.com/jaegertracing/jaeger-operator/pkg/util"
)

const (
	tmpWorkingDir = "/tmp/_certs"
)

type secret struct {
	name           string
	keyFileNameMap map[string]string
}

func (s secret) instanceName(jaeger *v1.Jaeger) string {
	// elasticsearch secret is hardcoded in es-operator https://jira.coreos.com/browse/LOG-326
	if s.name == esSecret.name {
		return esSecret.name
	}
	return fmt.Sprintf("%s-%s", jaeger.Name, s.name)
}

// master secret is used to generate other certs
var masterSecret = secret{
	name: "master-certs",
	keyFileNameMap: map[string]string{
		"ca":     "ca.crt",
		"ca-key": "ca.key",
	},
}

// es secret is used by Elasticsearch nodes
var esSecret = secret{
	name: "elasticsearch",
	keyFileNameMap: map[string]string{
		"elasticsearch.key": "elasticsearch.key",
		"elasticsearch.crt": "elasticsearch.crt",
		"logging-es.key":    "logging-es.key",
		"logging-es.crt":    "logging-es.crt",
		"admin-key":         "system.admin.key",
		"admin-cert":        "system.admin.crt",
		"admin-ca":          "ca.crt",
	},
}

// jaeger secret is used by jaeger components to talk to Elasticsearch
var jaegerSecret = secret{
	name: "jaeger-elasticsearch",
	keyFileNameMap: map[string]string{
		"ca":   "ca.crt",
		"key":  "user.jaeger.key",
		"cert": "user.jaeger.crt",
	},
}

// curator secret is used for index cleaner and rollover
var curatorSecret = secret{
	name: "curator",
	keyFileNameMap: map[string]string{
		"ca":   "ca.crt",
		"key":  "system.logging.curator.key",
		"cert": "system.logging.curator.crt",
	},
}

// ExtractSecrets assembles a set of secrets related to Elasticsearch
func (ed *ElasticsearchDeployment) ExtractSecrets() []corev1.Secret {
	return []corev1.Secret{
		createSecret(ed.Jaeger, masterSecret.instanceName(ed.Jaeger), getWorkingDirContents(getWorkingDir(ed.Jaeger), masterSecret.keyFileNameMap)),
		createSecret(ed.Jaeger, esSecret.instanceName(ed.Jaeger), getWorkingDirContents(getWorkingDir(ed.Jaeger), esSecret.keyFileNameMap)),
		createSecret(ed.Jaeger, jaegerSecret.instanceName(ed.Jaeger), getWorkingDirContents(getWorkingDir(ed.Jaeger), jaegerSecret.keyFileNameMap)),
		createSecret(ed.Jaeger, curatorSecret.instanceName(ed.Jaeger), getWorkingDirContents(getWorkingDir(ed.Jaeger), curatorSecret.keyFileNameMap)),
	}
}

// CreateCerts creates certificates for elasticsearch, jaeger and curator
// The cert generation is done by shell script. If the certificates are not present
// on the filesystem the operator injects them from secrets - this allows operator restarts.
// The script also re-generates expired certificates.
func (ed *ElasticsearchDeployment) CreateCerts() error {
	err := extractSecretsToFile(ed.Jaeger, ed.Secrets, masterSecret, esSecret, jaegerSecret, curatorSecret)
	if err != nil {
		return errors.Wrap(err, "failed to extract certificates from secrets to file")
	}
	return createESCerts(ed.CertScript, ed.Jaeger)
}

// CleanCerts removes certificates from local filesystem.
// Use this function in tests to clean resources
func (ed *ElasticsearchDeployment) CleanCerts() error {
	return os.RemoveAll(getWorkingDir(ed.Jaeger))
}

func extractSecretsToFile(jaeger *v1.Jaeger, secrets []corev1.Secret, s ...secret) error {
	secretMap := map[string]corev1.Secret{}
	for _, sec := range secrets {
		secretMap[sec.Name] = sec
	}
	for _, sec := range s {
		if secret, ok := secretMap[sec.instanceName(jaeger)]; ok {
			if err := extractSecretToFile(getWorkingDir(jaeger), secret.Data, sec); err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to extract secret %s", secret.Name))
			}
		}
	}
	return nil
}

func extractSecretToFile(workingDir string, data map[string][]byte, secret secret) error {
	for k, v := range secret.keyFileNameMap {
		if err := writeToFile(workingDir, v, data[k]); err != nil {
			return err

		}
	}
	return nil
}

func getWorkingDir(jaeger *v1.Jaeger) string {
	return filepath.Clean(fmt.Sprintf("%s/%s/%s/%s", tmpWorkingDir, jaeger.Namespace, jaeger.Name, jaeger.UID))
}

// createESCerts runs bash scripts which generates certificates
func createESCerts(script string, jaeger *v1.Jaeger) error {
	// #nosec   G204: Subprocess launching should be audited
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"NAMESPACE="+jaeger.Namespace,
		"WORKING_DIR="+getWorkingDir(jaeger),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithFields(log.Fields{
			"script": script,
			"out":    string(out)}).
			Error("Failed to create certificates")
		return fmt.Errorf("error running script %s: %v", script, err)
	}
	return nil
}

func createSecret(jaeger *v1.Jaeger, secretName string, data map[string][]byte) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: jaeger.Namespace,
			Labels: map[string]string{
				"app":                          "jaeger",
				"app.kubernetes.io/name":       secretName,
				"app.kubernetes.io/instance":   jaeger.Name,
				"app.kubernetes.io/component":  "es-secret",
				"app.kubernetes.io/part-of":    "jaeger",
				"app.kubernetes.io/managed-by": "jaeger-operator",
			},
			OwnerReferences: []metav1.OwnerReference{util.AsOwner(jaeger)},
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

func getWorkingDirContents(dir string, content map[string]string) map[string][]byte {
	c := map[string][]byte{}
	for secretKey, certName := range content {
		c[secretKey] = getDirFileContents(dir, certName)
	}
	return c
}

func getDirFileContents(dir, filePath string) []byte {
	return getFileContents(getFilePath(dir, filePath))
}

func getFilePath(dir, toFile string) string {
	return path.Join(dir, toFile)
}

func getFileContents(path string) []byte {
	if path == "" {
		return nil
	}
	contents, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil
	}
	return contents
}

func writeToFile(dir, file string, value []byte) error {
	// first check if file exists - we prefer what is on FS to revert users editing secrets
	path := getFilePath(dir, file)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(value)
	if err != nil {
		// remove the file on failure - it can be correctly created in the next iteration
		os.RemoveAll(path)
		return err
	}
	return nil
}
