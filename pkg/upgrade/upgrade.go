package upgrade

import (
	"context"
	"reflect"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// ManagedInstances finds all the Jaeger instances for the current operator and upgrades them, if necessary
func ManagedInstances(c client.Client) error {
	list := &v1.JaegerList{}
	opts := &client.ListOptions{}
	if err := c.List(context.Background(), opts, list); err != nil {
		return err
	}

	for _, j := range list.Items {
		jaeger, err := ManagedInstance(c, j)
		if err != nil {
			// nothing to do at this level, just go to the next instance
			continue
		}

		if !reflect.DeepEqual(jaeger, j) {
			// the CR has changed, store it!
			if err := c.Update(context.Background(), &jaeger); err != nil {
				log.WithFields(log.Fields{
					"instance":  jaeger.Name,
					"namespace": jaeger.Namespace,
				}).WithError(err).Error("failed to store the upgraded instance")
			}
		}
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given Jaeger instance to the current version
func ManagedInstance(client client.Client, jaeger v1.Jaeger) (v1.Jaeger, error) {
	if v, ok := versions[jaeger.Status.Version]; ok {
		// we don't need to run the upgrade function for the version 'v', only the next ones
		for n := v.next; n != nil; n = n.next {
			// performs the upgrade to version 'n'
			upgraded, err := n.upgrade(client, jaeger)
			if err != nil {
				log.WithFields(log.Fields{
					"instance":  jaeger.Name,
					"namespace": jaeger.Namespace,
					"to":        n.v,
				}).WithError(err).Warn("failed to upgrade managed instance")
				return jaeger, err
			}

			upgraded.Status.Version = n.v
			jaeger = upgraded
		}
	}

	return jaeger, nil
}
