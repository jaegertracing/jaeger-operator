// +build upgrade

package e2e

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

const (
	envUpgradeVersionKey = "UPGRADE_TEST_VERSION"
	upgradeTestTag       = "next"
)

type OperatorUpgradeTestSuite struct {
	suite.Suite
}

func TestOperatorUpgrade(t *testing.T) {
	suite.Run(t, new(OperatorUpgradeTestSuite))
}

func (suite *OperatorUpgradeTestSuite) SetupTest() {
	t = suite.T()
	var err error
	ctx, err = prepare(t)
	if err != nil {
		ctx.Cleanup()
		require.FailNow(t, "Failed in prepare")
	}
	addToFrameworkSchemeForSmokeTests(t)
	if err := simplest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
	fw = framework.Global
}

func (suite *OperatorUpgradeTestSuite) TearDownSuite() {
	handleSuiteTearDown()
}

func (suite *OperatorUpgradeTestSuite) TestUpgrade() {
	upgradeTestVersion := os.Getenv(envUpgradeVersionKey)
	versionRegexp := regexp.MustCompile(`^[\d]+\.[\d]+\.[\d]+`)
	require.Regexp(t, versionRegexp, upgradeTestVersion,
		"Invalid upgrade version, need to specify a version to upgrade in format X.Y.Z")
	createdJaeger := &v1.Jaeger{}
	key := types.NamespacedName{Name: "my-jaeger", Namespace: ctx.GetID()}
	fw.Client.Get(context.Background(), key, createdJaeger)
	deployment := &appsv1.Deployment{}
	fw.Client.Get(context.Background(), types.NamespacedName{Name: "jaeger-operator", Namespace: ctx.GetID()}, deployment)
	image := deployment.Spec.Template.Spec.Containers[0].Image
	image = strings.Replace(image, "latest", upgradeTestTag, 1)
	deployment.Spec.Template.Spec.Containers[0].Image = image
	t.Logf("Attempting to upgrade to version %s...", upgradeTestVersion)
	fw.Client.Update(context.Background(), deployment)
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		updatedJaeger := &v1.Jaeger{}
		key := types.NamespacedName{Name: "my-jaeger", Namespace: ctx.GetID()}
		if err := fw.Client.Get(context.Background(), key, updatedJaeger); err != nil {
			return true, err
		}
		if updatedJaeger.Status.Version == upgradeTestVersion {
			return true, nil
		}
		return false, nil

	})

	require.NoError(t, err, "upgrade e2e test failed")
}
