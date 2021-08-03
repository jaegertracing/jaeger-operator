# Releasing the Jaeger Operator for Kubernetes

Steps to release a new version of the Jaeger Operator:


1. Change the `versions.txt `so that it lists the target version of the Jaeger (if it is required). Don't touch the operator version it will be changed automatrically in the next ste.

2. Run `make prepare-release VERSION=1.25.0`, using the operator version that will be released.

3. Prepare a changelog since last release. 

4. Commit the changes and create a pull request:

   ```
   git commit -sm "Preparing release v1.25.0"
   ```

5. Once the changes above are merged and available in `master` tag it with the desired version, prefixed with `v`, eg. `v1.25.0`

    ```
    git checkout master
    git tag v1.25.0
    git push git@github.com:jaegertracing/jaeger-operator.git release/v1.25.0
    ```

6. The GitHub Workflow will take it from here, creating a GitHub release and publishing the images

    ```
    git pull git@github.com:jaegertracing/jaeger-operator.git master
    ```

7. After the release, PRs needs to be created against the Operator Hub Community Operators repositories:

    * One for the `upstream-community-operators`, used by OLM on Kubernetes. (https://github.com/k8s-operatorhub/community-operators)
    * One for the `community-operators` used by OpenShift. (https://github.com/redhat-openshift-ecosystem/community-operators-prod)

This can be done with the following steps:
- Update master `git pull git@github.com:jaegertracing/jaeger-operator.git master`
- Clone both repositories `upstream-community-operators` and `community-operators` 
- Run `make operatorhub`
  * If you have [`gh`](https://cli.github.com/) installed and configured, it will open the necessary PRs for you automatically.
  * If you don't have it, the branches will be pushed to `origin` and you should be able to open the PR from there

## Generating the changelog

- Get the `OAUTH_TOKEN` from (Github)[https://github.com/settings/tokens/new?description=GitHub%20Changelog%20Generator%20token] select `repo:status` scope.
- Run  `OAUTH_TOKEN=... make changelog`
- Remove the commits that are not relevant to users, like:
  * CI or testing-specific commits (e2e, unit test, ...)
  * bug fixes for problems that are not part of a release yet
  * version bumps for internal dependencies