# How to Contribute to the Jaeger Operator for Kubernetes

We'd love your help!

This project is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted.

We gratefully welcome improvements to documentation as well as to code.

## Getting Started

This project is a regular [Kubernetes Operator](https://coreos.com/operators/)  built using the Operator SDK. Refer to the Operator SDK documentation to understand the basic architecture of this operator.

### Installing the Operator SDK command line tool

Follow the installation guidelines from [Operator SDK GitHub page](https://github.com/operator-framework/operator-sdk) or run `make install-sdk`.

### Developing

As usual for operators following the Operator SDK in recent versions, the dependencies are managed using [`go modules`](https://golang.org/doc/go1.11#modules). Refer to that project's documentation for instructions on how to add or update dependencies.

The first step is to get a local Kubernetes instance up and running. The recommended approach is using `minikube`. Refer to the Kubernetes'  [documentation](https://kubernetes.io/docs/tasks/tools/install-minikube/) for instructions on how to install it.

Once `minikube` is installed, it can be started with:

```
minikube start
```

NOTE: Make sure to read the documentation to learn the performance switches that can be applied to your platform.

Once minikube has finished starting, get the Operator running:

```
make run
```

At this point, a Jaeger instance can be installed:

```
kubectl apply -f examples/simplest.yaml
kubectl get jaegers
kubectl get pods
```

To remove the instance:

```
kubectl delete -f examples/simplest.yaml
```

Tests should be simple unit tests and/or end-to-end tests. For small changes, unit tests should be sufficient, but every new feature should be accompanied with end-to-end tests as well. Tests can be executed with:

```
make test
```

NOTE: you can adjust the Docker image namespace by overriding the variable `NAMESPACE`, like: `make test NAMESPACE=quay.io/my-username`. The full Docker image name can be customized by overriding `BUILD_IMAGE` instead, like: `make test BUILD_IMAGE=quay.io/my-username/jaeger-operator:0.0.1`

#### Model changes

The Operator SDK generates the `pkg/apis/jaegertracing/v1/zz_generated.*.go` files via the command `make generate`. This should be executed whenever there's a model change (`pkg/apis/jaegertracing/v1/jaeger_types.go`)

#### Ingress configuration

Kubernetes comes with no ingress provider by default. For development purposes, when running `minikube`, the following command can be executed to install an ingress provider:

```
make ingress
```

This will install the `NGINX` ingress provider. It's recommended to wait for the ingress pods to be in the `READY` and `RUNNING` state before starting the operator. You can check it by running:

```
kubectl get pods -n ingress-nginx
```

To verify that it's working, deploy the `simplest.yaml` and check the ingress routes:

```
$ kubectl apply -f examples/simplest.yaml
jaeger.jaegertracing.io/simplest created
$ kubectl get ingress
NAME             HOSTS     ADDRESS          PORTS     AGE
simplest-query   *         192.168.122.69   80        12s
```

Accessing the provided "address" in your web browser should display the Jaeger UI.

#### Storage configuration

There are a set of templates under the `test` directory that can be used to setup an Elasticsearch and/or Cassandra cluster. Alternatively, the following commands can be executed to install it:

```
make es
make cassandra
```

#### Operator-Lifecycle-Manager Integration

The [Operator-Lifecycle-Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager/) can install, manage, and upgrade operators and their dependencies in a cluster.

With OLM, users can:

* Define applications as a single Kubernetes resource that encapsulates requirements and metadata
* Install applications automatically with dependency resolution or manually with nothing but kubectl
* Upgrade applications automatically with different approval policies

OLM also enforces some constraints on the components it manages in order to ensure a good user experience.

The Jaeger community provides and maintains a [ClusterServiceVersion (CSV) YAML](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/building-your-csv.md/) to integrate with OLM.

Starting from operator-sdk v0.5.0, one can generate and update CSVs based on the yaml files in the deploy folder.
The Jaeger CSV can be updated to version 1.9.0 with the following command:

```
$ operator-sdk generate csv --csv-version 1.9.0
INFO[0000] Generating CSV manifest version 1.9.0
INFO[0000] Create deploy/olm-catalog/jaeger-operator.csv.yaml 
INFO[0000] Create deploy/olm-catalog/_generated.concat_crd.yaml 
```

The generated CSV yaml should then be compared and used to update the deploy/olm-catalog/jaeger.clusterserviceversion.yaml file which represents the stable version copied to the operatorhub following each jaeger operator release. Once merged, the jaeger-operator.csv.yaml file should be removed.

The jaeger.clusterserviceversion.yaml file can then be tested with this command:

```
$ operator-sdk scorecard --cr-manifest examples/simplest.yaml --csv-path deploy/olm-catalog/jaeger.clusterserviceversion.yaml --init-timeout 30
Checking for existence of spec and status blocks in CR
Checking that operator actions are reflected in status
Checking that writing into CRs has an effect
Checking for CRD resources
Checking for existence of example CRs
Checking spec descriptors
Checking status descriptors
Basic Operator:
	Spec Block Exists: 1/1 points
	Status Block Exist: 1/1 points
	Operator actions are reflected in status: 0/1 points
	Writing into CRs has an effect: 1/1 points
OLM Integration:
	Owned CRDs have resources listed: 0/1 points
	CRs have at least 1 example: 1/1 points
	Spec fields with descriptors: 0/12 points
	Status fields with descriptors: N/A (depends on an earlier test that failed)

Total Score: 4/18 points
```

#### E2E tests

The whole set of end-to-end tests can be executed via:

```
$ make e2e-tests
```

The end-to-end tests are split into tags and can be executed in separate groups, such as:

```
$ make e2e-tests-smoke
```

Other targets include `e2e-tests-cassandra` and `e2e-tests-elasticsearch`. Refer to the `Makefile` for an up-to-date list of targets.

If you face issues like the one below, make sure you don't have any Jaeger instances (`kubectl get jaegers`) running nor Ingresses (`kubectl get ingresses`):

```
--- FAIL: TestSmoke (316.59s)
    --- FAIL: TestSmoke/smoke (316.55s)
        --- FAIL: TestSmoke/smoke/daemonset (115.54s)
...
...
            daemonset.go:30: timed out waiting for the condition
...
...
```

##### Kuttl E2E tests

There are some tests that uses [Kuttl](https://kuttl.dev/), those tests can be executed via:

```
$ make kuttl-e2e
```

You first need to install [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) in order to run the based kuttle e2e tests

#### Building [OCI Images](https://github.com/opencontainers/image-spec/blob/master/spec.md) for multiple arch (linux/arm64, linux/amd64)

OCI images could be built and published by [buildx](https://github.com/docker/buildx), it could be executed for local test via:

```
$ OPERATOR_VERSION=devel ./.ci/publish-images.sh
```

more arch support only need to change `--platform=linux/amd64,linux/arm64`

if we want to execute this in local env, need to setup buildx:

1. install docker cli plugin

```
$ export DOCKER_BUILDKIT=1
$ docker build --platform=local -o . git://github.com/docker/buildx
$ mkdir -p ~/.docker/cli-plugins
$ mv buildx ~/.docker/cli-plugins/docker-buildx
```
(via https://github.com/docker/buildx#with-buildx-or-docker-1903)

2. install qemu for multi arch

```
$ docker run --privileged --rm tonistiigi/binfmt --install all
```
(via https://github.com/docker/buildx#building-multi-platform-images)

3. create a builder

```
$ docker buildx create --use --name builder
```
