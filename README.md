[![Build Status][ci-img]][ci] [![Go Report Card][goreport-img]][goreport] [![Code Coverage][cov-img]][cov] [![GoDoc][godoc-img]][godoc] [![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/jaegertracing/jaeger-operator/badge)](https://securityscorecards.dev/viewer/?uri=github.com/jaegertracing/jaeger-operator)

# Jaeger Operator for Kubernetes

The Jaeger Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

## Getting started

Firstly, ensure an [ingress-controller is deployed](https://kubernetes.github.io/ingress-nginx/deploy/). When using `minikube`, you can use the `ingress` add-on: `minikube start --addons=ingress`

Then follow the Jaeger Operator [installation instructions](https://www.jaegertracing.io/docs/latest/operator/).

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

CRD-API documentation can be found [here](./docs/api.md).

## Compatibility matrix

See the compatibility matrix [here](./COMPATIBILITY.md).


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

Sometimes it is preferable to generate plain manifests files instead of running an operator in a cluster. `jaeger-operator generate` generates kubernetes manifests from a given CR. In this example we apply the manifest generated by [examples/simplest.yaml](https://raw.githubusercontent.com/jaegertracing/jaeger-operator/main/examples/simplest.yaml) to the namespace `jaeger-test`:

```bash
curl https://raw.githubusercontent.com/jaegertracing/jaeger-operator/main/examples/simplest.yaml | docker run -i --rm jaegertracing/jaeger-operator:main generate | kubectl apply -n jaeger-test -f -
```

It is recommended to deploy the operator instead of generating a static manifest.

## Jaeger V2 Operator

As the Jaeger V2 is released, it is decided that Jaeger V2 will deployed on Kubernetes using [OpenTelemetry Operator](https://github.com/open-telemetry/opentelemetry-operator). This will benefit both the users of Jaeger and OpenTelemetry. To use Jaeger V2 with OpenTelemetry Operator, the steps are as follows:

* Install the cert-manager in the existing cluster with the command:
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.1/cert-manager.yaml
```

Please verify all the resources (e.g., Pods and Deployments) are in a ready state in the `cert-manager` namespace.

* Install the OpenTelemetry Operator by running:
```bash
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
```

Please verify all the resources (e.g., Pods and Deployments) are in a ready state in the `opentelemetry-operator-system` namespace.

### Using Jaeger with in-memory storage

Once all the resources are ready, create a Jaeger instance as follows:
```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: jaeger-inmemory-instance
spec:
  image: jaegertracing/jaeger:latest
  ports:
  - name: jaeger
    port: 16686
  config:
    service:
      extensions: [jaeger_storage, jaeger_query]
      pipelines:
        traces:
          receivers: [otlp]    
          exporters: [jaeger_storage_exporter]
    extensions:
      jaeger_query:
        storage:
          traces: memstore
      jaeger_storage:
        backends:
          memstore:
            memory:
              max_traces: 100000
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    exporters:
      jaeger_storage_exporter:
        trace_storage: memstore
EOF
```

To use the in-memory storage ui for Jaeger V2, expose the pod, deployment or the service as follows:
```bash
kubectl port-forward deployment/jaeger-inmemory-instance-collector 8080:16686
```

Or

```bash
kubectl port-forward service/jaeger-inmemory-instance-collector 8080:16686
```

Once done, type `localhost:8080` in the browser to interact with the UI.

[Note] There's an ongoing development in OpenTelemetry Operator where users will be able to interact directly with the UI.

### Using Jaeger with database to store traces
To use Jaeger V2 with the supported database, it is mandatory to create database deployments and they should be in `ready` state [(ref)](https://www.jaegertracing.io/docs/2.0/storage/).

Create a Kubernetes Service that exposes the database pods enabling communication between the database and Jaeger pods.

This can be achieved by creating a service in two ways, first by creating it [manually](https://kubernetes.io/docs/concepts/services-networking/service/) or second by creating it using imperative command.

```bash
kubectl expose pods <pod-name> --port=<port-number> --name=<name-of-the-service>
```

Or

```bash
kubectl expose deployment <deployment-name> --port=<port-number> --name=<name-of-the-service>
```

After the service is created, add the name of the service as an endpoint in their respective config as follows:

* [Cassandra DB](https://github.com/jaegertracing/jaeger/blob/main/cmd/jaeger/config-cassandra.yaml):
```yaml
jaeger_storage:
  backends:
    some_storage:
      cassandra:
        connection:
          servers: [<name-of-the-service>]
```

* [ElasticSearch](https://github.com/jaegertracing/jaeger/blob/main/cmd/jaeger/config-elasticsearch.yaml):
```yaml
jaeger_storage:
  backends:
    some_storage:
      elasticseacrh:
        servers: [<name-of-the-service>]
```

Use the modified config to create Jaeger instance with the help of OpenTelemetry Operator.

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: jaeger-storage-instance # name of your choice
spec:
  image: jaegertracing/jaeger:latest
  ports:
  - name: jaeger
    port: 16686
  config:
    # modified config
EOF
```

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache 2.0 License](./LICENSE).

[ci-img]: https://github.com/jaegertracing/jaeger-operator/workflows/CI%20Workflow/badge.svg
[ci]: https://github.com/jaegertracing/jaeger-operator/actions
[cov-img]: https://codecov.io/gh/jaegertracing/jaeger-operator/branch/main/graph/badge.svg
[cov]: https://codecov.io/github/jaegertracing/jaeger-operator/
[goreport-img]: https://goreportcard.com/badge/github.com/jaegertracing/jaeger-operator
[goreport]: https://goreportcard.com/report/github.com/jaegertracing/jaeger-operator
[godoc-img]: https://godoc.org/github.com/jaegertracing/jaeger-operator?status.svg
[godoc]: https://godoc.org/github.com/jaegertracing/jaeger-operator/apis/v1#JaegerSpec
