package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestCreateCerts_ErrNoNamespace(t *testing.T) {
	err := createESCerts(certScript)
	assert.EqualError(t, err, "failed to get watch namespace: WATCH_NAMESPACE must be set")
}

func TestCreateCerts_ErrNoScript(t *testing.T) {
	os.Setenv("WATCH_NAMESPACE", "invalid.&*)(")
	defer os.Unsetenv("WATCH_NAMESPACE")
	err := createESCerts("invalid")
	assert.EqualError(t, err, "error running script invalid: exit status 127")
}

func TestCreateESSecrets(t *testing.T) {
	defer os.RemoveAll(workingDir)
	j := v1alpha1.NewJaeger("foo")
	os.Setenv("WATCH_NAMESPACE", "invalid.&*)(")
	defer os.Unsetenv("WATCH_NAMESPACE")
	fmt.Println(os.Getwd())
	err := createESCerts("../../scripts/cert_generation.sh")
	assert.NoError(t, err)
	sec := createESSecrets(j)
	assert.Equal(t, 4, len(sec))
	assert.Equal(t, []string{
		"master-certs",
		"elasticsearch",
		fmt.Sprintf("%s-%s", j.Name, jaegerSecret.name),
		fmt.Sprintf("%s-%s", j.Name, curatorSecret.name)},
		[]string{sec[0].Name, sec[1].Name, sec[2].Name, sec[3].Name})
	for _, s := range sec {
		if s.Name == fmt.Sprintf("%s-%s", j.Name, jaegerSecret.name) {
			ca, err := ioutil.ReadFile(workingDir + "/ca.crt")
			assert.NoError(t, err)
			assert.Equal(t, map[string][]byte{"ca": ca}, s.Data)
		}
	}
}

func TestCreteSecret(t *testing.T) {
	defer os.RemoveAll(workingDir)
	j := v1alpha1.NewJaeger("foo")
	j.Namespace = "myproject"
	s := createSecret(j, "bar", map[string][]byte{"foo": {}})
	assert.Equal(t, "bar", s.ObjectMeta.Name)
	assert.Equal(t, j.Namespace, s.ObjectMeta.Namespace)
	assert.Equal(t, j.Name, s.ObjectMeta.OwnerReferences[0].Name)
	assert.Equal(t, j.Name, s.ObjectMeta.OwnerReferences[0].Name)
	assert.Equal(t, map[string][]byte{"foo": {}}, s.Data)
	assert.Equal(t, v1.SecretTypeOpaque, s.Type)
}

func TestGetWorkingFileDirContent(t *testing.T) {
	defer os.RemoveAll(workingDir)
	err := os.MkdirAll(workingDir, os.ModePerm)
	assert.NoError(t, err)
	err = ioutil.WriteFile(workingDir+"/foobar", []byte("foo"), 0644)
	assert.NoError(t, err)
	b := getWorkingDirFileContents("foobar")
	assert.Equal(t, "foo", string(b))
}

func TestGetWorkingFileDirContent_EmptyPath(t *testing.T) {
	b := getWorkingDirFileContents("")
	assert.Nil(t, b)
}

func TestGetWorkingFileDirContent_FileDoesNotExists(t *testing.T) {
	b := getWorkingDirFileContents("jungle")
	assert.Nil(t, b)
}

func TestGetFileContet_EmptyPath(t *testing.T) {
	b := getFileContents("")
	assert.Nil(t, b)
}
