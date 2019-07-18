#!/usr/bin/env bash

set -eoux pipefail

export OPENSHIFT_VERSION=v3.9

function prepare() {
  sudo docker cp $(docker create docker.io/openshift/origin:$OPENSHIFT_VERSION):/bin/oc /usr/local/bin/oc
  oc cluster up --version=$OPENSHIFT_VERSION
  oc login -u system:admin
}

prepare

