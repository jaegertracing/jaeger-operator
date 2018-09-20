#!/bin/bash

echo "Uploading code coverage results"
bash <(curl -s https://codecov.io/bash)
