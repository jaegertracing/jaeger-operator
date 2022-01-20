package secrets

import (
	"io/ioutil"
	"log"
)

// GetToken reads the token from the given path and returns it
func GetToken(path string) string {
	content, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal(err)
	}

	return string(content)
}
