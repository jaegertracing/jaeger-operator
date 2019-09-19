# Releasing the Jaeger Operator for Kubernetes

1. Make sure you are using an operator-sdk newer than 0.10.0 (or master, if no released version exists)

1. Make sure the `versions.txt` file is up to date

1. Make sure the new version is present at `pkg/upgrade/versions.go`

1. Prepare a changelog and get it merged. A list of commits since the last release (`v1.8.0` in the following example) can be obtained via:

    ```
    $ git log --format="format:* %s" v1.8.0...HEAD
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
    git tag release/v1.6.1
    git push git@github.com:jaegertracing/jaeger-operator.git release/v1.6.1
    ```

1. Temporary step: run the release script manually, as the release workflow is temporarily disabled

1. Apply generated OLM catalog files to operatorhub.io

    * Clone the [operatorhub](https://github.com/operator-framework/community-operators) repo
    * Run `make operatorhub`
      - If you have [`hub`](https://hub.github.com/) installed and configured, it will open the necessary PRs for you automatically. Hint: `dnf install hub` works fine on Fedora.
      - If you don't have it, the branches will be pushed to `origin` and you should be able to open the PR from there
