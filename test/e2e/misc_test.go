// +build smoke

package e2e

import (
	"os"
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MiscTestSuite struct {
	suite.Suite
}

func(suite *MiscTestSuite) SetupSuite() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if (err != nil) {
		if ctx != nil {
			ctx.Cleanup()
		}
		require.FailNow(t, "Failed in prepare with: " + err.Error())
	}
	fw = framework.Global
	namespace, _ = ctx.GetNamespace()
	require.NotNil(t, namespace, "GetNamespace failed")

	addToFrameworkSchemeForSmokeTests(t)
}

func (suite *MiscTestSuite) TearDownSuite() {
	log.Info("Entering TearDownSuite()")
	ctx.Cleanup()
}

func TestMiscSuite(t *testing.T) {
	suite.Run(t, new(MiscTestSuite))
}

func (suite *MiscTestSuite) SetupTest() {
	t = suite.T()
}

// Make sure we're testing correct image
func (suite *MiscTestSuite) TestValidateBuildImage() {
	buildImage := os.Getenv("BUILD_IMAGE")
	require.NotEmptyf(t, buildImage, "BUILD_IMAGE must be defined")

	// If we did the normal test setup the operator will be in the current namespace.  If not we need to iterate
	// over all namespaces to find it, being sure to match on WATCH_NAMESPACE
	didSetup := len(noSetup) == 0
	if didSetup {
		validateBuildImageInTestNamespace(buildImage)
	} else {
		validateBuildImageInCluster(buildImage)
	}
}

func validateBuildImageInCluster(buildImage string) {
	emptyOptions := new(metav1.ListOptions)
	namespaces, err := fw.KubeClient.CoreV1().Namespaces().List(*emptyOptions)
	require.NoError(t, err)
	found := false
	for _, item := range namespaces.Items {
		imagesMap, err := getJaegerOperatorImages(fw.KubeClient, item.Name)
		require.NoError(t, err)

		if len(imagesMap) > 0 {
			watchNamespace, ok := imagesMap[buildImage]
			if ok {
				if len(watchNamespace) == 0 || strings.Contains(watchNamespace, item.Name) {
					found = true
					t.Logf("Using jaeger-operator image(s) %s\n", imagesMap)
					break
				}
			}
		}
	}
	require.Truef(t, found, "Could not find an operator with image %s", buildImage)
}

func validateBuildImageInTestNamespace(buildImage string) {
	imagesMap, err := getJaegerOperatorImages(fw.KubeClient, namespace)
	require.NoError(t, err)
	require.Len(t, imagesMap, 1, "Expected 1 deployed operator")
	_, ok := imagesMap[buildImage]
	require.Truef(t, ok, "Expected operator image %s not found in map %s\n", buildImage, imagesMap)
	t.Logf("Using jaeger-operator image(s) %s\n", imagesMap)
}
