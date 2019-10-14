package upgrade

import (
	"context"
	"reflect"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1"
)

// ManagedInstances finds all the Jaeger instances for the current operator and upgrades them, if necessary
func ManagedInstances(c client.Client) error {
	list := &v1.JaegerList{}
	identity := viper.GetString(v1.ConfigIdentity)
	opts := client.MatchingLabels(map[string]string{
		v1.LabelOperatedBy: identity,
	})
	if err := c.List(context.Background(), list, opts); err != nil {
		return err
	}

	for _, j := range list.Items {
		// this check shouldn't have been necessary, as I'd expect the list of items to come filtered out already
		// but apparently, at least the fake client used in the unit tests doesn't filter it out... so, let's double-check
		// that we indeed own the item
		owner := j.Labels[v1.LabelOperatedBy]
		if owner != identity {
			log.WithFields(log.Fields{
				"our-identity":   identity,
				"owner-identity": owner,
			}).Debug("skipping CR upgrade as we are not owners")
			continue
		}

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
