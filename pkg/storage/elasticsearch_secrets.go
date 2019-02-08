package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

const (
	workingDir = "/tmp/_working_dir"
	certScript = "./scripts/cert_generation.sh"
)

var secretCertificates = map[string]map[string]string{
	"master-certs": {
		"masterca":  "ca.crt",
		"masterkey": "ca.key",
	},
	"elasticsearch": {
		"elasticsearch.key": "elasticsearch.key",
		"elasticsearch.crt": "elasticsearch.crt",
		"logging-es.key":    "logging-es.key",
		"logging-es.crt":    "logging-es.crt",
		"admin-key":         "system.admin.key",
		"admin-cert":        "system.admin.crt",
		"admin-ca":          "ca.crt",
	},
	"jaeger-elasticsearch": {
		"ca":          "ca.crt",
		"jaeger-key":  "jaeger.user.key",
		"jaeger-cert": "jaeger.user.crt",
	},
}

func CreateESSecrets(jaeger *v1alpha1.Jaeger) []*v1.Secret {
	var secrets []*v1.Secret
	for name, content := range secretCertificates {
		c := map[string][]byte{}
		for secretKey, certName := range content {
			c[secretKey] = getWorkingDirFileContents(certName)
		}
		s := createSecret(jaeger, name, c)
		secrets = append(secrets, s)
	}
	return secrets
}

// 1. extract certificates from secret map (e.g. master-certs) and put them into a file but if the file is not found it returns no error
// 2. run script to generate certs (perhapts this overrides the extracted secrets?)
// 3. the secret is updated but only master-certs. It was done in CreateOrUpdateSecret
func CreateESCerts() error {
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return fmt.Errorf("failed to get watch namespace: %v", err)
	}
	// #nosec   G204: Subprocess launching should be audited
	cmd := exec.Command("bash", certScript)
	cmd.Env = append(os.Environ(),
		"NAMESPACE="+namespace,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		logrus.WithFields(logrus.Fields{
			"script": certScript,
			"out":    string(out)}).
			Error("Failed to create certificates")
		return fmt.Errorf("error running script: %v", err)
	}
	return nil
}

func createSecret(jaeger *v1alpha1.Jaeger, secretName string, data map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName,
			Namespace:       jaeger.Namespace,
			OwnerReferences: []metav1.OwnerReference{asOwner(jaeger)},
		},
		Type: v1.SecretTypeOpaque,
		Data: data,
	}
}

func asOwner(jaeger *v1alpha1.Jaeger) metav1.OwnerReference {
	b := true
	return metav1.OwnerReference{
		APIVersion: jaeger.APIVersion,
		Kind:       jaeger.Kind,
		Name:       jaeger.Name,
		UID:        jaeger.UID,
		Controller: &b,
	}
}

func getWorkingDirFileContents(filePath string) []byte {
	return getFileContents(getWorkingDirFilePath(filePath))
}

func getWorkingDirFilePath(toFile string) string {
	return path.Join(workingDir, toFile)
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
