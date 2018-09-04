#!/bin/bash

echo "Uploading code coverage results"
bash <(curl -s https://codecov.io/bash)

if [ "${TRAVIS_BRANCH}" = "master" -a "${TRAVIS_PULL_REQUEST}" = "false" ]; then
  echo "Releasing..."
  make docker
  echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
  export BUILD_IMAGE_TEST=${BUILD_IMAGE}
  export BUILD_IMAGE="jaegertracing/jaeger-operator:1.6"
  docker tag ${BUILD_IMAGE_TEST} ${BUILD_IMAGE}
  docker push ${BUILD_IMAGE}
fi
