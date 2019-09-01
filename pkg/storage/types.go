package storage

import (
	"strings"
)

// ValidTypes returns the list of valid storage types
func ValidTypes() []string {
	return []string{
		"memory",
		"kafka",
		"elasticsearch",
		"cassandra",
		"badger",
	}
}

// OptionsPrefix returns the options prefix associated with the supplied storage type
func OptionsPrefix(storageType string) string {
	if strings.EqualFold(storageType, "elasticsearch") {
		return "es"
	}
	return storageType
}
