package storage

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestCreteCerts(t *testing.T) {
	err := CreateESCerts()
	assert.EqualError(t, err, "failed to get watch namespace: WATCH_NAMESPACE must be set")
}

func TestCreteSecret(t *testing.T) {
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

func TestCreteESSecrets(t *testing.T) {
	j := v1alpha1.NewJaeger("foo")
	sec := CreateESSecrets(j)
	assert.Equal(t, len(secretCertificates), len(sec))
	for _, s := range sec {
		_, ok := secretCertificates[s.Name]
		assert.True(t, ok)
		if s.Name == "jaeger-elasticsearch" {
			assert.Equal(t, map[string][]byte{"ca": nil, "jaeger-key": nil, "jaeger-cert": nil}, s.Data)
		}
	}
}

func TestGetWorkingFileDirContent(t *testing.T) {
	err := ioutil.WriteFile(workingDir+"/foobar", []byte("foo"), 0644)
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
