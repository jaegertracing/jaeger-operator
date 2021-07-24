#!/bin/bash


COMMUNITY_OPERATORS_REPOSITORY="k8s-operatorhub/community-operators"
UPSTREAM_REPOSITORY="redhat-openshift-ecosystem/community-operators-prod"
LOCAL_REPOSITORIES_PATH=${LOCAL_REPOSITORIES_PATH:-"$(dirname $(dirname $(pwd)))"}


if [[ ! -d "${LOCAL_REPOSITORIES_PATH}/${COMMUNITY_OPERATORS_REPOSITORY}" ]]; then
    echo "${LOCAL_REPOSITORIES_PATH}/${COMMUNITY_OPERATORS_REPOSITORY} doesn't exist, aborting."
    exit 1
fi

if [[ ! -d "${LOCAL_REPOSITORIES_PATH}/${UPSTREAM_REPOSITORY}" ]]; then
    echo "${LOCAL_REPOSITORIES_PATH}/${UPSTREAM_REPOSITORY} doesn't exist, aborting."
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

for dest in ${COMMUNITY_OPERATORS_REPOSITORY} ${UPSTREAM_REPOSITORY}; do
    cd "${LOCAL_REPOSITORIES_PATH}/${dest}"
    git remote | grep upstream > /dev/null
    if [[ $? != 0 ]]; then
        echo "Cannot find a remote named 'upstream'. Adding one."
        git remote add upstream git@github.com:${dest}.git
    fi

    git fetch -q upstream
    git checkout -q main
    git rebase -q upstream/main

    mkdir -p "${dest}/operators/jaeger/${VERSION}"

    cp "${OLD_PWD}/${PKG_FILE}" "${dest}/operators/jaeger/${DEST_PKG_FILE}"
    cp "${OLD_PWD}/${CSV_FILE}" "${dest}/operators/jaeger/${VERSION}/${DEST_CSV_FILE}"
    cp "${OLD_PWD}/${CRD_FILE}" "${dest}/operators/jaeger/${VERSION}"

    git checkout -q -b Update-Jaeger-to-${VERSION}
    if [[ $? != 0 ]]; then
        echo "Cannot switch to the new branch Update-Jaeger-${dest}-to-${VERSION}. Aborting"
        exit 1
    fi

    git add ${dest}
    git commit -sqm "Update Jaeger to v${VERSION}"


    command -v gh > /dev/null
    if [[ $? != 0 ]]; then
        echo "'gh' command not found, can't submit the PR on your behalf."
        break
    fi
    tmpfile=$(mktemp /tmp/Update-Jaeger-to-${VERSION}.XXX)
    echo "Update Jaeger to v${VERSION}" > ${tmpfile}
    curl https://raw.githubusercontent.com/${dest}/main/docs/pull_request_template.md >> "${tmpfile}"

    echo "Submitting PR on your behalf via 'hub'"
    gh pr create --title  "Update Jaeger to v${VERSION}" --body-file "${tmpfile}"
    rm ${tmpfile}
done

cd ${OLD_PWD}
echo "Completed."
