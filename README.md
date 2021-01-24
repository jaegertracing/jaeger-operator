
[![Build Status][ci-img]][ci] [![Go Report Card][goreport-img]][goreport] [![Code Coverage][cov-img]][cov] [![GoDoc][godoc-img]][godoc]

# Jaeger Operator for Kubernetes

This branch is a work in progress. Contains an initial work for jaeger operator v2, this operator deploys Jaeger v2 which is based on OpenTelemetry.

## Getting started

To install the operator in an existing cluster, make sure you have [Open Telemetry Operator installed](https://github.com/open-telemetry/opentelemetry-operator) and run:
```
make install run
```
Once the `jaeger-operator` deployment is ready, create an Jaeger instance, like:

```console
$ kubectl apply -f - <<EOF
apiVersion: jaegertracing.io/v2
kind: Jaeger
metadata:
  name: simple-prod
spec:
  strategy: production
EOF
```

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## Testing

With an existing cluster (such as `minikube`), run:
```
USE_EXISTING_CLUSTER=true make test
```

Tests can also be run without an existing cluster. For that, install [`kubebuilder`](https://book.kubebuilder.io/quick-start.html#installation). In this case, the tests will bootstrap `etcd` and `kubernetes-api-server` for the tests. Run against an existing cluster whenever possible, though.

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
