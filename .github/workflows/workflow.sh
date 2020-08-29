#! /usr/bin/env bash

set -ue -o pipefail
trap "echo >&2 script '${BASH_SOURCE[0]}' failed" ERR

SCRIPT=`readlink -e "${BASH_SOURCE[0]}"`
SCRIPT_DIR=`dirname "$SCRIPT"`
cd "$SCRIPT_DIR"/../..
unset SCRIPT SCRIPT_DIR

GOFMT_OUTPUT=`gofmt -l -s .`
if [ -n "$GOFMT_OUTPUT" ]; then
	gofmt -d -s .
	false
fi
unset GOFMT_OUTPUT

go vet ./...
go test -bench=. ./...
go install ./...
