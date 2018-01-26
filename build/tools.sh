#!/usr/bin/env bash
#set -x
scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

error_exit () {
    >&2 echo "${1}"
    if [[ $# -gt 1 ]]
    then
        exit ${2}
    else
        exit 1
    fi
}

copy_tools () {
    if [[ -z "$GOPATH" ]]; then
        error_exit "GOPATH env var should be set..."
    fi
    for tool in $@; do
        tool="${tool%%/...*}"
        if [[ ! -x $GOPATH/bin/${tool##*/} ]]; then
            echo "$GOPATH/bin/${tool##*/} doesn't exist. We copy it from build tools folder to $GOPATH/bin"
            arch=$(uname -m)
            if [[ ! -d ${scriptDir}/tools/${arch} ]]; then
                error_exit "No binaries found for arch:${arch}"
            fi
            cp ${scriptDir}/tools/${arch}/${tool##*/} $GOPATH/bin
        fi
    done
}

go get -u -v $@
if [[ $? -ne 0 ]]; then
    echo "go get command failed for tools building : we'll use saved binaries instead."
    copy_tools $@
fi