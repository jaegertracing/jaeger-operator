# Releasing the Jaeger Operator for Kubernetes

## Generating the changelog

- Get the `OAUTH_TOKEN` from [Github](https://github.com/settings/tokens/new?description=GitHub%20Changelog%20Generator%20token), select `repo:status` scope.
- Run  `OAUTH_TOKEN=... make changelog`
- Remove the commits that are not relevant to users, like:
  * CI or testing-specific commits (e2e, unit test, ...)
  * bug fixes for problems that are not part of a release yet
  * version bumps for internal dependencies

## Releasing

Steps to release a new version of the Jaeger Operator:


1. Change the `versions.txt `so that it lists the target version of the Jaeger (if it is required). **Don't touch the operator version**: it will be changed automatically in the next step.

2. Confirm that `MIN_KUBERNETES_VERSION` and `MIN_OPENSHIFT_VERSION` in the `Makefile` are still up-to-date, and update them if required.

2. Run `OPERATOR_VERSION=1.30.0 make prepare-release`, using the operator version that will be released.

3. Run the E2E tests in OpenShift as described in [the CONTRIBUTING.md](CONTRIBUTING.md#an-external-cluster-like-openshift) file. The tests will be executed automatically in Kubernetes by the GitHub Actions CI later.

4. Prepare a changelog since last release.

4. Update the release manager schedule.

5. Commit the changes and create a pull request:

   ```sh
   git commit -sm "Preparing release v1.30.0"
   ```

5. Once the changes above are merged and available in `main` tag it with the desired version, prefixed with `v`, eg. `v1.30.0`

    ```sh
    git checkout main
    git tag v1.30.0
    git push git@github.com:jaegertracing/jaeger-operator.git v1.30.0
    ```

6. The GitHub Workflow will take it from here, creating a GitHub release and publishing the images

7. After the release, PRs needs to be created against the Operator Hub Community Operators repositories:

    * One for the [upstream-community-operators](https://github.com/k8s-operatorhub/community-operators), used by OLM on Kubernetes.
    * One for the [community-operators](https://github.com/redhat-openshift-ecosystem/community-operators-prod) used by OpenShift.

This can be done with the following steps:
- Update main `git pull git@github.com:jaegertracing/jaeger-operator.git main`
- Clone both repositories `upstream-community-operators` and `community-operators`
- Run `make operatorhub`
  * If you have [`gh`](https://cli.github.com/) installed and configured, it will open the necessary PRs for you automatically.
  * If you don't have it, the branches will be pushed to `origin` and you should be able to open the PR from there

## Note
After the PRs have been made it must be ensured that:
- Images listed in the ClusterServiceVersion (CSV) have a versions tag [#1682](https://github.com/jaegertracing/jaeger-operator/issues/1682)
- No `bundle` folder is included in the release
- No foreign CRs like prometheus are in the manifests

## Release managers

The operator should be released within a week after the [Jaeger release](https://github.com/jaegertracing/jaeger/blob/main/RELEASE.md#release-managers).

| Version | Release Manager                                          |
|---------| -------------------------------------------------------- |
| 1.61.0  | [Israel Blancas](https://github.com/iblancasa)           |
| 1.62.0  | [Ruben Vargas](https://github.com/rubenvp8510)           |
| 1.63.0  | [Benedikt Bongartz](https://github.com/frzifus)          |
| 1.64.0  | [Pavol Loffay](https://github.com/pavolloffay)           |
