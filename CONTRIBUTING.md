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

### Local run

Build the manifests, install the CRD and run the operator as a local process:
```
$ make manifests install run
```

## Contributing

Your contribution is welcome! For it to be accepted, we have a few standards that must be followed.

### New features

Before starting the development of a new feature, please create an issue and discuss it with the project maintainers. Features should come with documentation and enough tests (unit and/or end-to-end).

### Bug fixes

Every bug fix should be accompanied with a unit test, so that we can prevent regressions.

### Documentation, typos, ...

They are mostly welcome!