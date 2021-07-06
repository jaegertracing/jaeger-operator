
[![Build Status][ci-img]][ci] [![Go Report Card][goreport-img]][goreport] [![Code Coverage][cov-img]][cov] [![GoDoc][godoc-img]][godoc]

# Jaeger Operator for Kubernetes

The Jaeger Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

## Getting started

Firstly, ensure an [ingress-controller is deployed](https://kubernetes.github.io/ingress-nginx/deploy/). When using `minikube`, you can use the `ingress` add-on: `minikube start --addons=ingress`

To install the operator, run:
```
kubectl create namespace observability
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/crds/jaegertracing.io_jaegers_crd.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/service_account.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role_binding.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/operator.yaml
```

The operator will activate extra features if given cluster-wide permissions. To enable that, run:
```
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role.yaml
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role_binding.yaml
```

Note that you'll need to download and customize the `cluster_role_binding.yaml` if you are using a namespace other than `observability`. You probably also want to download and customize the `operator.yaml`, setting the env var `WATCH_NAMESPACE` to have an empty value, so that it can watch for instances across all namespaces.

Once the `jaeger-operator` deployment in the namespace `observability` is ready, create a Jaeger instance, like:

```
kubectl apply -n observability -f - <<EOF
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: simplest
EOF
```

This will create a Jaeger instance named `simplest`. The Jaeger UI is served via the `Ingress`, like:

```console
$ kubectl get -n observability ingress
NAME             HOSTS     ADDRESS          PORTS     AGE
simplest-query   *         192.168.122.34   80        3m
```

In this example, the Jaeger UI is available at http://192.168.122.34.

The official documentation for the Jaeger Operator, including all its customization options, are available under the main [Jaeger Documentation](https://www.jaegertracing.io/docs/latest/operator/).

## Compatibility matrix

The following table shows the compatibility of jaeger operator with different components, in this particular case we shows Kubernetes and Strimzi operator compatibility


| Jaeger Operator | Kubernetes           | Strimzi Operator   |
|-----------------|----------------------|---------------------
| v1.23           | v1.19+               | v0.19, v0.20       |
| v1.22           | v1.18 to v1.20       | v0.19              |


### Jaeger Operator vs. Jaeger

The Jaeger Operator follows the same versioning as the operand (Jaeger) up to the minor part of the version. For example, the Jaeger Operator v1.22.2 tracks Jaeger 1.22.0. The patch part of the version indicates the patch level of the operator itself, not that of Jaeger. Whenever a new patch version is released for Jaeger, we'll release a new patch version of the operator.

### Jaeger Operator vs. Kubernetes

We strive to be compatible with the widest range of Kubernetes versions as possible, but some changes to Kubernetes itself require us to break compatibility with older Kubernetes versions, be it because of code imcompatibilities, or in the name of maintainability.

Our promise is that we'll follow what's common practice in the Kubernetes world and support N-2 versions, based on the release date of the Jaeger Operator.

For instance, when we released v1.22.0, the latest Kubernetes version was v1.20.5. As such, the minimum version of Kubernetes we support for Jaeger Operator v1.22.0 is v1.18 and we tested it with up to 1.20.

The Jaeger Operator *might* work on versions outside of the given range, but when opening new issues, please make sure to test your scenario on a supported version.


### Jaeger Operator vs. Strimzi Operator

We maintain compatibility with a set of tested Strimzi operator versions, but some changes in Strimzi operator require us to break compatibility with older versions.

The jaeger Operator *might* work on other untested versions of Strimzi Operator, but when opening new issues, please make sure to test your scenario on a supported version.


## (experimental) Generate Kubernetes manifest file

Sometimes it is preferable to generate plain manifests files instead of running an operator in a cluster. `jaeger-operator generate` generates kubernetes manifests from a given CR. In this example we apply the manifest generated by [examples/simplest.yaml](https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/examples/simplest.yaml) to the namespace `jaeger-test`:

```bash
curl https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/examples/simplest.yaml | docker run -i --rm jaegertracing/jaeger-operator:master generate | kubectl apply -n jaeger-test -f -
```

It is recommended to deploy the operator instead of generating a static manifest.

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## License
  
[Apache 2.0 License](./LICENSE).

[ci-img]: https://github.com/jaegertracing/jaeger-operator/workflows/CI%20Workflow/badge.svg
[ci]: https://github.com/jaegertracing/jaeger-operator/actions
[cov-img]: https://codecov.io/gh/jaegertracing/jaeger-operator/branch/master/graph/badge.svg
[cov]: https://codecov.io/github/jaegertracing/jaeger-operator/
[goreport-img]: https://goreportcard.com/badge/github.com/jaegertracing/jaeger-operator
[goreport]: https://goreportcard.com/report/github.com/jaegertracing/jaeger-operator
[godoc-img]: https://godoc.org/github.com/jaegertracing/jaeger-operator?status.svg
[godoc]: https://godoc.org/github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1#JaegerSpec
