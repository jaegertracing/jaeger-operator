package secrets

import (
	"io/ioutil"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// GetToken reads the token from the given path and returns it
func GetToken(path string) string {
	content, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		logrus.Errorln("Something failed during reading the token:", err)
		return ""
	}

	return string(content)
}
