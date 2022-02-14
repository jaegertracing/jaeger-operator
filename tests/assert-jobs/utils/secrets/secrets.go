package secrets

import (
	"io/ioutil"
	"log"
	"path/filepath"
)

// GetToken reads the token from the given path and returns it
func GetToken(path string) string {
	content, err := ioutil.ReadFile(filepath.Clean(path))

	if err != nil {
		log.Fatal(err)
	}

	return string(content)
}
