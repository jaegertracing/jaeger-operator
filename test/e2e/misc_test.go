// +build smoke

package e2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	log "github.com/sirupsen/logrus"
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
	imagesMap, err := getJaegerOperatorImages(fw.KubeClient)
	require.NoError(t, err)

	// TODO update test to deal with multiple installed operators if necessary
	require.Len(t, imagesMap, 1, "Expected 1 deployed operator")

	_, ok := imagesMap[buildImage]
	require.Truef(t, ok, "Expected operator image %s not found in map %s\n", buildImage, imagesMap)
	t.Logf("Using jaeger-operator image(s) %s\n", imagesMap)
}
