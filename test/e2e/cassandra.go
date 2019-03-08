package e2e

import (
	goctx "context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// Cassandra runs a test with Cassandra as the backing storage
func Cassandra(t *testing.T) {
	ctx := prepare(t)
	defer ctx.Cleanup()

	if err := cassandraTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func cassandraTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	cleanupOptions := &framework.CleanupOptions{TestContext: ctx, Timeout: timeout, RetryInterval: retryInterval}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	j := &v1.Jaeger{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Jaeger",
			APIVersion: "jaegertracing.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "with-cassandra",
			Namespace: namespace,
		},
		Spec: v1.JaegerSpec{
			Strategy: "allInOne",
			Storage: v1.JaegerStorageSpec{
				Type:    "cassandra",
				Options: v1.NewOptions(map[string]interface{}{"cassandra.servers": cassandraServiceName, "cassandra.keyspace": "jaeger_v1_datacenter1"}),
				CassandraCreateSchema: v1.JaegerCassandraCreateSchemaSpec{
					Datacenter: "datacenter1",
				},
			},
		},
	}

	log.Infof("passing %v", j)
	err = f.Client.Create(goctx.TODO(), j, cleanupOptions)
	if err != nil {
		return err
	}

	err = WaitForJob(t, f.KubeClient, namespace, "with-cassandra-cassandra-schema-job", retryInterval, timeout)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "with-cassandra", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	jaegerPod, err := GetPod(namespace, "with-cassandra", "jaegertracing/all-in-one", f.KubeClient)
	if err != nil {
		return err
	}
	portForw, closeChan, err := CreatePortForward(namespace, jaegerPod.Name, []string{"0:16686", "0:14268"}, f.KubeConfig)
	if err != nil {
		return err
	}
	defer portForw.Close()
	defer close(closeChan)
	ports, err := portForw.GetPorts()
	if err != nil {
		return err
	}
	return SmokeTest(fmt.Sprintf("http://localhost:%d/api/traces", ports[0].Local), fmt.Sprintf("http://localhost:%d/api/traces", ports[1].Local), "foobar", retryInterval, timeout)
}
