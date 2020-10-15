#!/usr/bin/env bash
RE='\([0-9]\+\)[.]\([0-9]\+\)[.]\([0-9]\+\)\([0-9A-Za-z-]*\)'
MAJOR=$(echo ${1} | sed -e "s#${RE}#\1#")
MINOR=$(echo ${1} | sed -e "s#${RE}#\2#")
PATCH=$(echo ${1} | sed -e "s#${RE}#\3#")
PATCH=$(( $PATCH + 1 ))
echo "${MAJOR}.${MINOR}.${PATCH}"

