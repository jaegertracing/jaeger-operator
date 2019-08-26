#!/bin/bash

make install
RT=$?
if [ ${RT} != 0 ]; then
    echo "Failed to install the operator dependencies."
    exit ${RT}
fi
