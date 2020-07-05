#! /usr/bin/env bash

set -ue -o pipefail
trap "echo >&2 script '${BASH_SOURCE[0]}' failed" ERR

SCRIPT=`readlink -e "${BASH_SOURCE[0]}"`
SCRIPT_DIR=`dirname "$SCRIPT"`
cd "$SCRIPT_DIR"/../..

! gofmt -l -s . | grep -q ''
go vet ./...
go test ./...
go install ./...
