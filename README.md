
[![Build Status][ci-img]][ci] [![Go Report Card][goreport-img]][goreport] [![Code Coverage][cov-img]][cov] [![GoDoc][godoc-img]][godoc]

# Jaeger Operator for Kubernetes

The Jaeger Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

## Getting started
1) Create the namespace: 
```
kubectl create namespace observability
```
2) Load the CRD based on your version:
- Kubernetes 1.12+:
```
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/crds/jaegertracing.io_jaegers_crd.yaml
```
- Kubernetes 1.11-: 
```
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/crds/jaegertracing.io_jaegers_crd_ocp311.yaml
```

3) To install the operator, run:
```
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/crds/jaegertracing.io_jaegers_crd.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/service_account.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role_binding.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/operator.yaml
```

4) The operator will activate extra features if given cluster-wide permissions. To enable that, run:
```
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role.yaml
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role_binding.yaml
```

Note that you'll need to download and customize the `cluster_role_binding.yaml` if you are using a namespace other than `observability`. You probably also want to download and customize the `operator.yaml`, setting the env var `WATCH_NAMESPACE` to have an empty value, so that it can watch for instances across all namespaces.

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

[ci-img]: https://github.com/jaegertracing/jaeger-operator/workflows/CI%20Workflow/badge.svg
[ci]: https://github.com/jaegertracing/jaeger-operator/actions
[cov-img]: https://codecov.io/gh/jaegertracing/jaeger-operator/branch/master/graph/badge.svg
[cov]: https://codecov.io/github/jaegertracing/jaeger-operator/
[goreport-img]: https://goreportcard.com/badge/github.com/jaegertracing/jaeger-operator
[goreport]: https://goreportcard.com/report/github.com/jaegertracing/jaeger-operator
[godoc-img]: https://godoc.org/github.com/jaegertracing/jaeger-operator?status.svg
[godoc]: https://godoc.org/github.com/jaegertracing/jaeger-operator/pkg/apis/jaegertracing/v1#JaegerSpec
