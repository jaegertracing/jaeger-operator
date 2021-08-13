#!/usr/bin/env bash

PARAMS=""
DEPLOYMENT=""
while (( "$#" )); do
  case "$1" in
    -n|--namespace)
      if [ -n "$2" ] && [ ${2:0:1} != "-" ]; then
        NAMESPACE=$2
        shift 2
      else
        echo "Error: Argument for $1 is missing" >&2
        exit 1
      fi
      ;;
    -*|--*=) # unsupported flags
      echo "Error: Unsupported flag $1" >&2
      exit 1
      ;;
    *) # preserve positional arguments
      PARAMS="$PARAMS $1"
      shift
      ;;
  esac
done

kubectl annotate --overwrite deployments ${DEPLOYMENT} "sidecar.jaegertracing.io/inject"="false"
