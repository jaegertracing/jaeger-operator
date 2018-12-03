#!/bin/bash

set -e

python .travis/import-order-cleanup.py -o $1 -t $(git ls-files "*\.go" | grep -v -e vendor)
