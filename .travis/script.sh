#!/bin/bash

make ci BUILD_IMAGE=${BUILD_IMAGE}
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to build the operator."
    exit ${RT}
fi
