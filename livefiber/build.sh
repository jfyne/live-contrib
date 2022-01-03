#!/usr/bin/env bash

set -e

if ! command -v embedmd &> /dev/null
then
    GO111MODULE=off go get github.com/campoy/embedmd
fi
embedmd -w README.md
