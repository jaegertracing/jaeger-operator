# How to Contribute to the Jaeger Operator for Kubernetes

We'd love your help!

This project is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted.

We gratefully welcome improvements to documentation as well as to code.

This project is a regular [Kubernetes Operator](https://coreos.com/operators/)  built using the Operator SDK. Refer to the Operator SDK documentation to understand the basic architecture of this operator.

## Installing the Operator SDK command line tool

Follow the installation guidelines from [Operator SDK GitHub page](https://github.com/operator-framework/operator-sdk)

## Developing

As usual for operators following the Operator SDK in recent versions, the dependencies are managed using [`go modules`](https://golang.org/doc/go1.11#modules). Refer to that project's documentation for instructions on how to add or update dependencies.

The first step is to get a local Kubernetes instance up and running. The recommended approach for development is using `minikube` with *ingress* enabled. Refer to the Kubernetes'  [documentation](https://kubernetes.io/docs/tasks/tools/install-minikube/) for instructions on how to install it.

Once `minikube` is installed, it can be started with:
```sh
minikube start --addons=ingress
```

NOTE: Make sure to read the documentation to learn the performance switches that can be applied to your platform.

Log into docker (or another image registry):
```sh
docker login --username <dockerusername>
```

Once minikube has finished starting, get the Operator running:
```sh
make cert-manager
IMG=docker.io/$USER/jaeger-operator:latest make generate bundle docker push deploy
```

NOTE: If your registry username is not the same as $USER, modify the previous command before executing it.  Also change *docker.io* if you are using a different image registry.

At this point, a Jaeger instance can be installed:
```sh
kubectl apply -f examples/simplest.yaml
kubectl get jaegers
kubectl get pods
```

To verify the Jaeger instance is running, execute *minikube ip* and open that address in a browser, or follow the steps below
```sh
export MINIKUBE_IP=`minikube ip`
curl http://{$MINIKUBE_IP}/api/services
```
NOTE: you may have to execute the *curl* command twice to get a non-empty result

Tests should be simple unit tests and/or end-to-end tests. For small changes, unit tests should be sufficient, but every new feature should be accompanied with end-to-end tests as well. Tests can be executed with:
```sh
make test
```

#### Cleaning up
To remove the instance:
```sh
kubectl delete -f examples/simplest.yaml
```



#### Model changes

The Operator SDK generates the `pkg/apis/jaegertracing/v1/zz_generated.*.go` files via the command `make generate`. This should be executed whenever there's a model change (`pkg/apis/jaegertracing/v1/jaeger_types.go`)

### Storage configuration

There are a set of templates under the `test` directory that can be used to setup an Elasticsearch and/or Cassandra cluster. Alternatively, the following commands can be executed to install it:

```sh
make es
make cassandra
```

### Operator-Lifecycle-Manager Integration

The [Operator-Lifecycle-Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager/) can install, manage, and upgrade operators and their dependencies in a cluster.

With OLM, users can:

* Define applications as a single Kubernetes resource that encapsulates requirements and metadata
* Install applications automatically with dependency resolution or manually with nothing but kubectl
* Upgrade applications automatically with different approval policies

OLM also enforces some constraints on the components it manages in order to ensure a good user experience.

The Jaeger community provides and maintains a [ClusterServiceVersion (CSV) YAML](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md) to integrate with OLM.

Starting from operator-sdk v0.5.0, one can generate and update CSVs based on the yaml files in the deploy folder.
The Jaeger CSV can be updated to version 1.9.0 with the following command:

```sh
$ operator-sdk generate csv --csv-version 1.9.0
INFO[0000] Generating CSV manifest version 1.9.0
INFO[0000] Create deploy/olm-catalog/jaeger-operator.csv.yaml
INFO[0000] Create deploy/olm-catalog/_generated.concat_crd.yaml
```

The generated CSV yaml should then be compared and used to update the `deploy/olm-catalog/jaeger.clusterserviceversion.yaml` file which represents the stable version copied to the operatorhub following each jaeger operator release. Once merged, the `jaeger-operator.csv.yaml` file should be removed.

The `jaeger.clusterserviceversion.yaml` file can then be tested with this command:
```sh
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

## E2E tests

### Requisites

Before running the E2E tests you need to install:

* [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation): a tool for running local Kubernetes clusters
* [KUTTL](https://kuttl.dev/docs/cli.html#setup-the-kuttl-kubectl-plugin): a tool to run the Kubernetes tests


### Runing the E2E tests

#### Using KIND cluster
The whole set of end-to-end tests can be executed via:

```sh
$ make run-e2e-tests
```

The end-to-end tests are split into tags and can be executed in separate groups, such as:

```sh
$ make run-e2e-tests-examples
```

Other targets include `run-e2e-tests-cassandra` and `run-e2e-tests-elasticsearch`. You can list them running:
```sh
$ make e2e-test-suites
```

**Note**: there are some variables you need to take into account in order to
improve your experience running the E2E tests.

| Variable name     | Description                                         | Example usage                      |
|-------------------|-----------------------------------------------------|------------------------------------|
| KUTTL_OPTIONS     | Options to pass directly to the KUTTL call          | KUTTL_OPTIONS="--test es-rollover" |
| E2E_TESTS_TIMEOUT | Timeout for each step in the E2E tests. In seconds  | E2E_TESTS_TIMEOUT=500              |
| USE_KIND_CLUSTER  | Start a KIND cluster to run the E2E tests           | USE_KIND_CLUSTER=true              |
| KIND_KEEP_CLUSTER | Not remove the KIND cluster after running the tests | KIND_KEEP_CLUSTER=true             |

Also, you can enable/disable the installation of the different operators needed
to run the tests:
| Variable name  | Description                                 | Example usage       |
|----------------|---------------------------------------------|---------------------|
| JAEGER_OLM     | Jaeger Operator was installed using OLM     | JAEGER_OLM=true     |
| KAFKA_OLM      | Kafka Operator was installed using OLM      | KAFKA_OLM=true      |
| PROMETHEUS_OLM | Prometheus Operator was installed using OLM | PROMETHEUS_OLM=true |

#### An external cluster (like OpenShift)
The commands from the previous section are valid when running the E2E tests in an
external cluster like OpenShift, minikube or other Kubernetes environment. The only
difference are:
* You need to log in your Kubernetes cluster before running the E2E tests
* You need to provide the `USE_KIND_CLUSTER=false` parameter when calling `make`

For instance, to run the `examples` E2E test suite in OpenShift, the command is:
```sh
$ make run-e2e-tests-examples USE_KIND_CLUSTER=false
```

### Developing new E2E tests

E2E tests are located under `tests/e2e`. Each folder is associated to an E2E test suite. The
Tests are developed using KUTTL. Before developing a new test, [learn how KUTTL test works](https://kuttl.dev/docs/what-is-kuttl.html).

To add a new suite, it is needed to create a new folder with the name of the suite under `tests/e2e`.

Each suite folder contains:
* `Makefile`: describes the rules associated to rendering the files needed for your tests and run the tests
* `render.sh`: renders all the files needed for your tests (or to skip them)
* A folder per test to run

When the test are rendered, each test folder is copied to `_build`. The files generated
by `render.sh` are created under `_build/<test name>`.

##### Makefile
The `Makefile` file must contain two rules:

```Makefile
render-e2e-tests-<suite name>: set-assert-e2e-img-name
	./tests/e2e/<suite name>/render.sh

run-e2e-tests-<suite name>: TEST_SUITE_NAME=<suite name>
run-e2e-tests-<suite name>: run-suite-tests
```

Where `<suite name>` is the name of your E2E test suite. Your E2E test suite
will be automatically indexed in the `run-e2e-tests` Makefile target.

##### render.sh

This file renders all the YAML files that are part of the E2E test. The `render.sh`
file must start with:

```bash
#!/bin/bash

source $(dirname "$0")/../render-utils.sh
```

The `render-utils.sh` file contains multiple functions to make easier to develop E2E tests and reuse logic. You can go to it and review the documentation of each one of the functions to
understand their parameters and effects.

#### Building [OCI Images](https://github.com/opencontainers/image-spec/blob/master/spec.md) for multiple arch (linux/arm64, linux/amd64)

OCI images could be built and published by [buildx](https://github.com/docker/buildx), it could be executed for local test via:

```sh
$ OPERATOR_VERSION=devel ./.ci/publish-images.sh
```

more arch support only need to change `--platform=linux/amd64,linux/arm64`

if we want to execute this in local env, need to setup buildx:

1. install docker cli plugin

```sh
$ export DOCKER_BUILDKIT=1
$ docker build --platform=local -o . git://github.com/docker/buildx
$ mkdir -p ~/.docker/cli-plugins
$ mv buildx ~/.docker/cli-plugins/docker-buildx
```
(via https://github.com/docker/buildx#with-buildx-or-docker-1903)

2. install qemu for multi arch

```sh
$ docker run --privileged --rm tonistiigi/binfmt --install all
```
(via https://github.com/docker/buildx#building-multi-platform-images)

3. create a builder

```sh
$ docker buildx create --use --name builder
```
