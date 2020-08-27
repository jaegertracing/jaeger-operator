#!/bin/bash

if [ -z ${COMMUNITY_OPERATORS_REPOSITORY} ]; then
    COMMUNITY_OPERATORS_REPOSITORY="$(dirname $(dirname $(pwd)))/operator-framework/community-operators"
    echo "COMMUNITY_OPERATORS_REPOSITORY not set, using ${COMMUNITY_OPERATORS_REPOSITORY}"
fi

if [ ! -d ${COMMUNITY_OPERATORS_REPOSITORY} ]; then
    echo "${COMMUNITY_OPERATORS_REPOSITORY} doesn't exist, aborting."
    exit 1
fi

OLD_PWD=$(pwd)
VERSION=$(grep operator= versions.txt | awk -F= '{print $2}')
PKG_FILE=deploy/olm-catalog/jaeger-operator/jaeger-operator.package.yaml
CSV_FILE=deploy/olm-catalog/jaeger-operator/manifests/jaeger-operator.clusterserviceversion.yaml
CRD_FILE=deploy/crds/jaegertracing.io_jaegers_crd.yaml

# once we get a clarification on the following item, we might not need to have different file names
# https://github.com/operator-framework/community-operators/issues/701
DEST_PKG_FILE=jaeger.package.yaml
DEST_CSV_FILE=jaeger.v${VERSION}.clusterserviceversion.yaml

cd "${COMMUNITY_OPERATORS_REPOSITORY}"

git remote | grep upstream > /dev/null
if [ $? != 0 ]; then
    echo "Cannot find a remote named 'upstream'. Adding one."
    git remote add upstream git@github.com:operator-framework/community-operators.git
fi

git fetch -q upstream
git checkout -q master
git rebase -q upstream/master

for dest in upstream-community-operators community-operators; do
    mkdir -p "${COMMUNITY_OPERATORS_REPOSITORY}/${dest}/jaeger/${VERSION}"

    cp "${OLD_PWD}/${PKG_FILE}" "${COMMUNITY_OPERATORS_REPOSITORY}/${dest}/jaeger/${DEST_PKG_FILE}"
    cp "${OLD_PWD}/${CSV_FILE}" "${COMMUNITY_OPERATORS_REPOSITORY}/${dest}/jaeger/${VERSION}/${DEST_CSV_FILE}"
    cp "${OLD_PWD}/${CRD_FILE}" "${COMMUNITY_OPERATORS_REPOSITORY}/${dest}/jaeger/${VERSION}"

    git checkout -q master

    git checkout -q -b Update-Jaeger-${dest}-to-${VERSION}
    if [ $? != 0 ]; then
        echo "Cannot switch to the new branch Update-Jaeger-${dest}-to-${VERSION}. Aborting"
        exit 1
    fi

    git add "${COMMUNITY_OPERATORS_REPOSITORY}/${dest}/"
    git commit -sqm "Update Jaeger ${dest} to v${VERSION}"
    git push -q

    command -v hub > /dev/null
    if [ $? != 0 ]; then
        echo "'hub' command not found, can't submit the PR on your behalf."
        break
    fi

    tmpfile=$(mktemp /tmp/Update-Jaeger-${dest}-to-${VERSION}.XXX)
    cat > ${tmpfile} <<- EOM
Update Jaeger ${dest} to v${VERSION}

Thanks submitting your Operator. Please check below list before you create your Pull Request.
*************************************************
**Flat operator directory structure is obsolete from 23-rd of October 2019, only nested directory structure will be accepted.**
*************************************************

### New Submissions

* [x] Has you operator [nested directory structure](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#create-a-bundle)?
* [x] Have you selected the Project *Community Operator Submissions* in your PR on the right-hand menu bar?
* [x] Are you familiar with our [contribution guidelines](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md)?
* [x] Have you [packaged and deployed](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md) your Operator for Operator Framework?
* [x] Have you tested your Operator with all Custom Resource Definitions?
* [x] Have you tested your Operator in all supported [installation modes](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md#operator-metadata)?
* [x] Is your submission [signed](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#sign-your-work)?

### Updates to existing Operators

* [x] Is your new CSV pointing to the previous version with the replaces property?
* [x] Is your new CSV referenced in the [appropriate channel](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#bundle-format) defined in the package.yaml ?
* [ ] Have you tested an update to your Operator when deployed via OLM?
* [x] Is your submission [signed](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#sign-your-work)?

### Your submission should not

* [x] Modify more than one operator
* [x] Modify an Operator you don't own
* [x] Rename an operator - please remove and add with a different name instead
* [x] Submit operators to both upstream-community-operators and community-operators at once
* [x] Modify any files outside the above mentioned folders
* [x] Contain more than one commit. **Please squash your commits.**

### Operator Description must contain (in order)

1. [x] Description about the managed Application and where to find more information
2. [x] Features and capabilities of your Operator and how to use it
3. [x] Any manual steps about potential pre-requisites for using your Operator

### Operator Metadata should contain

* [x] Human readable name and 1-liner description about your Operator
* [x] Valid [category name](https://github.com/operator-framework/community-operators/blob/master/docs/required-fields.md#categories)<sup>1</sup>
* [x] One of the pre-defined [capability levels](https://github.com/operator-framework/operator-courier/blob/4d1a25d2c8d52f7de6297ec18d8afd6521236aa2/operatorcourier/validate.py#L556)<sup>2</sup>
* [x] Links to the maintainer, source code and documentation
* [x] Example templates for all Custom Resource Definitions intended to be used
* [x] A quadratic logo

Remember that you can preview your CSV [here](https://operatorhub.io/preview).

--

<sup>1</sup> If you feel your Operator does not fit any of the pre-defined categories, file a PR against this repo and explain your need

<sup>2</sup> For more information see [here](https://github.com/operator-framework/operator-sdk/blob/master/doc/images/operator-capability-level.svg)
EOM

    echo "Submitting PR on your behalf via 'hub'"
    hub pull-request -F ${tmpfile}
    rm ${tmpfile}
done

cd ${OLD_PWD}
echo "Completed."
