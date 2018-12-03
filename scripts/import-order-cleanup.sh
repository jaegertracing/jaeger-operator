#!/bin/bash

set -e

python scripts/import-order-cleanup.py -o $1 -t $(git ls-files "*\.go" | grep -v -e vendor)
