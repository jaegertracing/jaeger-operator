package e2e

import (
	"testing"

	f "github.com/operator-framework/operator-sdk/pkg/test"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestMain(m *testing.M) {
	f.MainEntry(m)
}
