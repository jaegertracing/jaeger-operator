#!/bin/bash
#
# Get the name to use for the E2E assert job
#
ASSERT_JOB_TAG=$(cat build-assert-job 2> /dev/null)
if [ $? != 0 ] || [ "$ASSERT_JOB_TAG" = "" ]; then
    echo "$ASSERT_IMG"
else
    echo "$ASSERT_JOB_TAG"
fi
