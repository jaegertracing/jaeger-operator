# How to Contribute to the Jaeger Operator

We'd love your help!

This project is [Apache 2.0 licensed](LICENSE) and accepts contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted.

We gratefully welcome improvements to documentation as well as to code.

## Getting Started

This project is a regular [Kubernetes Operator](https://coreos.com/operators/)  built using the Operator SDK. Refer to the Operator SDK documentation to understand the basic architecture of this operator.

### Workflow

It is recommended to follow the ["GitHub Workflow"](https://guides.github.com/introduction/flow/). When using [GitHub's CLI](https://github.com/cli/cli), here's how it typically looks like:

```
$ gh repo fork github.com/jaegertracing/jaeger-operator
$ git checkout -b your-feature-branch
# do your changes
$ git commit -sam "Add feature X"
$ gh pr create
```

### Pre-requisites
* Install [Go](https://golang.org/doc/install).
* Have a Kubernetes cluster ready for development. We recommend `minikube` or `kind`.
* This operator depends on [OpenTelemetry Operator](https://github.com/open-telemetry/opentelemetry-operator/blob/master/README.md), so first you need to install it 

### Local run

Build the manifests, install the CRD and run the operator as a local process:
```
$ make manifests install run
```

## Testing

With an existing cluster (such as `minikube`), run:
```
USE_EXISTING_CLUSTER=true make test
```

Tests can also be run without an existing cluster. For that, install [`kubebuilder`](https://book.kubebuilder.io/quick-start.html#installation). In this case, the tests will bootstrap `etcd` and `kubernetes-api-server` for the tests. Run against an existing cluster whenever possible, though.

### End to end tests

To run the end-to-end tests, you'll need [`kind`](https://kind.sigs.k8s.io) and [`kuttl`](https://kuttl.dev). Refer to their documentation for installation instructions.

Once they are installed, the tests can be executed with `make prepare-e2e`, which will build an image to use with the tests, followed by `make e2e`. Each call to the `e2e` target will setup a fresh `kind` cluster, making it safe to be executed multiple times with a single `prepare-e2e` step.

The tests are located under `tests/e2e` and are written to be used with `kuttl`. Refer to their documentation to understand how tests are written.

## Contributing

Your contribution is welcome! For it to be accepted, we have a few standards that must be followed.

### New features

Before starting the development of a new feature, please create an issue and discuss it with the project maintainers. Features should come with documentation and enough tests (unit and/or end-to-end).

### Bug fixes

Every bug fix should be accompanied with a unit test, so that we can prevent regressions.

### Documentation, typos, ...

They are mostly welcome!