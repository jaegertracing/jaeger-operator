
[![Build Status][ci-img]][ci] [![Go Report Card][goreport-img]][goreport] [![Code Coverage][cov-img]][cov] [![GoDoc][godoc-img]][godoc]

# Jaeger Operator for Kubernetes

The Jaeger Operator is an implementation of a [Kubernetes Operator](https://coreos.com/operators/).

## Getting started

To install the operator, run:
```
kubectl create namespace observability
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/crds/jaegertracing_v1_jaeger_crd.yaml
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/service_account.yaml
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role.yaml
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role_binding.yaml
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/operator.yaml
```

Once the `jaeger-operator` deployment in the namespace `observability` is ready, create a Jaeger instance, like:

```
kubectl apply -f - <<EOF
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: simplest
EOF
```

This will create a Jaeger instance named `simplest`. The Jaeger UI is served via the `Ingress`, like:

```console
$ kubectl get ingress
NAME             HOSTS     ADDRESS          PORTS     AGE
simplest-query   *         192.168.122.34   80        3m
```

In this example, the Jaeger UI is available at http://192.168.122.34.

The official documentation for the Jaeger Operator, including all its customization options, are available under the main [Jaeger Documentation](https://www.jaegertracing.io/docs/latest/operator/).

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## License
  
[Apache 2.0 License](./LICENSE).

[ci-img]: https://travis-ci.org/jaegertracing/jaeger-operator.svg?branch=master
[ci]: https://travis-ci.org/jaegertracing/jaeger-operator
[cov-img]: https://codecov.io/gh/jaegertracing/jaeger-operator/branch/master/graph/badge.svg
[cov]: https://codecov.io/github/jaegertracing/jaeger-operator/
[goreport-img]: https://goreportcard.com/badge/github.com/jaegertracing/jaeger-operator
[goreport]: https://goreportcard.com/report/github.com/jaegertracing/jaeger-operator
[godoc-img]: https://godoc.org/github.com/jaegertracing/jaeger-operator?status.svg
[godoc]: https://godoc.org/github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1#JaegerSpec
