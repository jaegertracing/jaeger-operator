package storage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCreateCerts_ErrNoScript(t *testing.T) {
	err := createESCerts("invalid", "", "")
	assert.EqualError(t, err, "error running script invalid: exit status 127")
}

func TestCreateESSecrets(t *testing.T) {
	defer os.RemoveAll(tmpWorkingDir + "/foo")
	j := v1.NewJaeger("foo")
	err := createESCerts("../../scripts/cert_generation.sh", tmpWorkingDir+"/foo", "")
	assert.NoError(t, err)
	sec := ESSecrets(j)
	assert.Equal(t, []string{
		masterSecret.instanceName(j),
		"elasticsearch",
		jaegerSecret.instanceName(j),
		curatorSecret.instanceName(j)},
		[]string{sec[0].Name, sec[1].Name, sec[2].Name, sec[3].Name})
	for _, s := range sec {
		if s.Name == jaegerSecret.instanceName(j) {
			ca, err := ioutil.ReadFile(tmpWorkingDir + "/foo/ca.crt")
			assert.NoError(t, err)
			assert.Equal(t, map[string][]byte{"ca": ca}, s.Data)
		}
	}
}

func TestCreateSecret(t *testing.T) {
	j := v1.NewJaeger("foo")
	j.Namespace = "myproject"
	s := createSecret(j, "bar", map[string][]byte{"foo": {}})
	assert.Equal(t, "bar", s.ObjectMeta.Name)
	assert.Equal(t, j.Namespace, s.ObjectMeta.Namespace)
	assert.Equal(t, j.Name, s.ObjectMeta.OwnerReferences[0].Name)
	assert.Equal(t, j.Name, s.ObjectMeta.OwnerReferences[0].Name)
	assert.Equal(t, map[string][]byte{"foo": {}}, s.Data)
	assert.Equal(t, corev1.SecretTypeOpaque, s.Type)
}

func TestGetWorkingFileDirContent(t *testing.T) {
	defer os.RemoveAll(tmpWorkingDir)
	err := os.MkdirAll(tmpWorkingDir, os.ModePerm)
	assert.NoError(t, err)
	err = ioutil.WriteFile(tmpWorkingDir+"/foobar", []byte("foo"), 0644)
	assert.NoError(t, err)
	b := getDirFileContents(tmpWorkingDir, "foobar")
	assert.Equal(t, "foo", string(b))
}

func TestGetWorkingFileDirContent_EmptyPath(t *testing.T) {
	b := getDirFileContents("", "")
	assert.Nil(t, b)
}

func TestGetWorkingFileDirContent_FileDoesNotExists(t *testing.T) {
	b := getDirFileContents("", "jungle")
	assert.Nil(t, b)
}

func TestGetFileContent_EmptyPath(t *testing.T) {
	b := getFileContents("")
	assert.Nil(t, b)
}

func TestExtractSecretsToFile(t *testing.T) {
	defer os.RemoveAll(tmpWorkingDir)
	j := v1.NewJaeger("houdy")
	j.Namespace = "bar"
	content := "115dasrez"
	err := extractSecretsToFile(
		j,
		[]corev1.Secret{{ObjectMeta: metav1.ObjectMeta{Name: "houdy-sec"}, Data: map[string][]byte{"ca": []byte(content)}}},
		secret{name: "sec", keyFileNameMap: map[string]string{"ca": "ca.crt"}},
	)
	assert.NoError(t, err)
	ca, err := ioutil.ReadFile(tmpWorkingDir + "/bar/houdy/ca.crt")
	assert.NoError(t, err)
	assert.Equal(t, []byte(content), ca)
}

func TestExtractSecretsToFile_FileExists(t *testing.T) {
	defer os.RemoveAll(tmpWorkingDir)
	content := "115dasrez"
	err := os.MkdirAll(tmpWorkingDir+"/bar/houdy", os.ModePerm)
	assert.NoError(t, err)
	err = ioutil.WriteFile(tmpWorkingDir+"/bar/houdy/ca.crt", []byte(content), os.ModePerm)
	assert.NoError(t, err)

	j := v1.NewJaeger("houdy")
	j.Namespace = "bar"
	err = extractSecretsToFile(
		j,
		[]corev1.Secret{{ObjectMeta: metav1.ObjectMeta{Name: "houdy-sec"}, Data: map[string][]byte{"ca": []byte("should not be there")}}},
		secret{name: "sec", keyFileNameMap: map[string]string{"ca": "ca.crt"}},
	)
	assert.NoError(t, err)
	ca, err := ioutil.ReadFile(tmpWorkingDir + "/bar/houdy/ca.crt")
	assert.NoError(t, err)
	assert.Equal(t, []byte(content), ca)
}
