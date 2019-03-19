package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCreateESSecrets(t *testing.T) {
	err := CreateESCerts(v1.NewJaeger("foo"), []corev1.Secret{})
	assert.EqualError(t, err, "error running script ./scripts/cert_generation.sh: exit status 127")
}

func TestCreateESSecrets_internal(t *testing.T) {
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

func TestWriteToWorkingDir(t *testing.T) {
	_, testFile, _, _ := runtime.Caller(0)
	defer os.RemoveAll(os.TempDir() + "/foo")
	tests := []struct {
		dir  string
		file string
		err  string
	}{
		{
			dir: "/foo", file: "", err: "mkdir /foo: permission denied",
		},
		{
			dir: "/root", file: "bla", err: "open /root/bla: permission denied",
		},
		{
			// file exists
			dir: path.Dir(testFile), file: path.Base(testFile),
		},
		{
			// write to file
			dir: os.TempDir(), file: "foo",
		},
	}
	for _, test := range tests {
		err := writeToFile(test.dir, test.file, []byte("random"))
		if test.err != "" {
			assert.EqualError(t, err, test.err)
		} else {
			assert.NoError(t, err)
			stat, err := os.Stat(fmt.Sprintf("%s/%s", test.dir, test.file))
			assert.NoError(t, err)
			assert.Equal(t, test.file, stat.Name())
		}
	}
}
