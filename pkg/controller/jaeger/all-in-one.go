package jaeger

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/jaegertracing/jaeger-operator/pkg/account"
	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
	"github.com/jaegertracing/jaeger-operator/pkg/configmap"
	"github.com/jaegertracing/jaeger-operator/pkg/deployment"
	"github.com/jaegertracing/jaeger-operator/pkg/ingress"
	"github.com/jaegertracing/jaeger-operator/pkg/inject"
	"github.com/jaegertracing/jaeger-operator/pkg/route"
	"github.com/jaegertracing/jaeger-operator/pkg/storage"
)

type allInOneController struct {
	ctx    context.Context
	jaeger *v1alpha1.Jaeger
}

func newAllInOneController(ctx context.Context, jaeger *v1alpha1.Jaeger) *allInOneController {
	return &allInOneController{
		ctx:    ctx,
		jaeger: jaeger,
	}
}

func (c *allInOneController) Create() []runtime.Object {
	logrus.Debugf("Creating all-in-one for '%v'", c.jaeger.Name)

	dep := deployment.NewAllInOne(c.jaeger)
	os := []runtime.Object{}

	// add all service accounts
	for _, acc := range account.Get(c.jaeger) {
		os = append(os, acc)
	}

	// add the deployments
	os = append(os, inject.OAuthProxy(c.jaeger, dep.Get()))

	ds := deployment.NewAgent(c.jaeger).Get()
	if nil != ds {
		os = append(os, ds)
	}

	// add the services
	for _, svc := range dep.Services() {
		os = append(os, svc)
	}

	// add the routes/ingresses
	if viper.GetString("platform") == v1alpha1.FlagPlatformOpenShift {
		if q := route.NewQueryRoute(c.jaeger).Get(); nil != q {
			os = append(os, q)
		}
	} else {
		if q := ingress.NewQueryIngress(c.jaeger).Get(); nil != q {
			os = append(os, q)
		}
	}

	// add the config map
	cm := configmap.NewUIConfig(c.jaeger).Get()
	if nil != cm {
		os = append(os, cm)
	}

	return os
}

func (c *allInOneController) Update() []runtime.Object {
	logrus.Debug("Update isn't available for all-in-one")
	return []runtime.Object{}
}

func (c *allInOneController) Dependencies() []batchv1.Job {
	return storage.Dependencies(c.jaeger)
}
