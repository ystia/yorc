#!/usr/bin/env bash
#set -x

error_exit () {
    >&2 echo "${1}"
    if [[ $# -gt 1 ]]
    then
        exit ${2}
    else
        exit 1
    fi
}


if [[ -z "$GOROOT" ]]; then
    error_exit "GOROOT env var should be set..."
fi

if [[ -z "$GOPATH" ]]; then
    error_exit "GOPATH env var should be set..."
fi

for tool in $@; do
    #Suppress trailing /... in url if any
    tool="${tool%%/...*}"
    if [[ ! -x $GOPATH/bin/${tool##*/} ]]; then
        error_exit "Tool not found $GOPATH/bin/${tool##*/} doesn't exist. This could be fixed by running 'make tools'"
    fi
done
