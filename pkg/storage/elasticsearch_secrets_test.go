package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

func TestCreateESSecretsError(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: t.Name()})
	es := &ElasticsearchDeployment{Jaeger: j, CertScript: "/foo"}
	err := es.CleanCerts()
	require.NoError(t, err)
	defer es.CleanCerts()
	err = es.CreateCerts()
	assert.EqualError(t, err, "error running script /foo: exit status 127")
}

func TestCreateESSecrets(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: t.Name()})
	es := &ElasticsearchDeployment{Jaeger: j, CertScript: "../../scripts/cert_generation.sh"}
	err := es.CleanCerts()
	require.NoError(t, err)
	defer es.CleanCerts()
	err = es.CreateCerts()
	assert.NoError(t, err)
	sec := es.ExtractSecrets()
	assert.Equal(t, []string{
		masterSecret.instanceName(j),
		esSecret.instanceName(j),
		jaegerSecret.instanceName(j),
		curatorSecret.instanceName(j)},
		[]string{sec[0].Name, sec[1].Name, sec[2].Name, sec[3].Name})
	for _, s := range sec {
		if s.Name == jaegerSecret.instanceName(j) {
			ca, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/ca.crt", tmpWorkingDir, j.Name))
			assert.NoError(t, err)
			key, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/user.jaeger.key", tmpWorkingDir, j.Name))
			assert.NoError(t, err)
			cert, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/user.jaeger.crt", tmpWorkingDir, j.Name))
			assert.NoError(t, err)
			assert.Equal(t, map[string][]byte{"ca": ca, "key": key, "cert": cert}, s.Data)
		}
	}
}

func TestCreateSecret(t *testing.T) {
	j := v1.NewJaeger(types.NamespacedName{Name: "foo"})
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
	dir := "/tmp/_" + t.Name()
	defer os.RemoveAll(dir)
	err := os.MkdirAll(dir, os.ModePerm)
	assert.NoError(t, err)
	err = ioutil.WriteFile(dir+"/foobar", []byte("foo"), 0644)
	assert.NoError(t, err)
	b := getDirFileContents(dir, "foobar")
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
	j := v1.NewJaeger(types.NamespacedName{Name: t.Name()})
	caFile := fmt.Sprintf("%s/%s/ca.crt", tmpWorkingDir, j.Name)
	defer os.Remove(caFile)
	content := "115dasrez"
	err := extractSecretsToFile(
		j,
		[]corev1.Secret{{ObjectMeta: metav1.ObjectMeta{Name: j.Name + "-sec"}, Data: map[string][]byte{"ca": []byte(content)}}},
		secret{name: "sec", keyFileNameMap: map[string]string{"ca": "ca.crt"}},
	)
	assert.NoError(t, err)
	ca, err := ioutil.ReadFile(caFile)
	assert.NoError(t, err)
	assert.Equal(t, []byte(content), ca)
}

func TestExtractSecretsToFile_Err(t *testing.T) {
	err := extractSecretToFile("/root", map[string][]byte{"foo": {}}, secret{keyFileNameMap: map[string]string{"foo": "foo"}})
	assert.EqualError(t, err, "open /root/foo: permission denied")
}

func TestExtractSecretsToFile_FileExists(t *testing.T) {
	defer os.RemoveAll(tmpWorkingDir + "/bar/houdy")
	content := "115dasrez"
	err := os.MkdirAll(tmpWorkingDir+"/bar/houdy", os.ModePerm)
	assert.NoError(t, err)
	err = ioutil.WriteFile(tmpWorkingDir+"/bar/houdy/ca.crt", []byte(content), os.ModePerm)
	assert.NoError(t, err)

	j := v1.NewJaeger(types.NamespacedName{Name: "houdy"})
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
