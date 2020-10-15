// +build upgrade

package e2e

import (
	"context"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"strings"
	"testing"
)

const EnvUpdateVersionKey = "UPDATE_TEST_VERSION"
const upgradeTestTag = "next"

func TestOperatorUpgrade(t *testing.T) {

	upgradeTestVersion := os.Getenv(EnvUpdateVersionKey)
	t.Log(upgradeTestVersion)

	ctx, err := prepare(t)
	if err != nil {
		ctx.Cleanup()
		require.FailNow(t, "Failed in prepare")
	}
	defer ctx.Cleanup()
	addToFrameworkSchemeForSmokeTests(t)
	if err := simplest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
	fw = framework.Global
	createdJaeger := &v1.Jaeger{}
	key := types.NamespacedName{Name: "my-jaeger", Namespace: ctx.GetID()}
	fw.Client.Get(context.Background(), key, createdJaeger)
	deployment := &appsv1.Deployment{}
	fw.Client.Get(context.Background(), types.NamespacedName{Name: "jaeger-operator", Namespace: ctx.GetID()}, deployment)
	image := deployment.Spec.Template.Spec.Containers[0].Image
	image = strings.Replace(image, "latest", upgradeTestTag, 1)
	deployment.Spec.Template.Spec.Containers[0].Image = image
	fw.Client.Update(context.Background(), deployment)
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
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
