# Releasing the Jaeger Operator for Kubernetes

1. Update Jaeger version in `versions.txt`

1. Make sure the new version is present at `pkg/upgrade/versions.go`

1. Prepare a changelog since last release and get it merged:

    ```
    $ OAUTH_TOKEN=... make changelog
    ```

1. Test!

    ```
    export BUILD_IMAGE_TEST="${USER}/jaeger-operator:latest"
    export BUILD_IMAGE="${BUILD_IMAGE_TEST}"
    make all
    ```

1. Tag and push

    ```
    git checkout master ## it's only possible to release from master for now!
    git tag release/v1.16.0
    git push git@github.com:jaegertracing/jaeger-operator.git release/v1.16.0
    ```

1. Apply generated OLM catalog files to operatorhub.io

    * Clone the [operatorhub](https://github.com/operator-framework/community-operators) repo
    * Run `make operatorhub`
      - If you have [`hub`](https://hub.github.com/) installed and configured, it will open the necessary PRs for you automatically. Hint: `dnf install hub` works fine on Fedora.
      - If you don't have it, the branches will be pushed to `origin` and you should be able to open the PR from there
