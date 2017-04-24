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

if [[ ! -e "${scriptDir}/consul" ]]; then
    cd ${scriptDir}
    consulVersion=$(grep consul_version ../versions.yaml | awk '{print $2}')

    zipName="consul_${consulVersion}_$(go env GOHOSTOS)_$(go env GOHOSTARCH).zip"
    wget "https://releases.hashicorp.com/consul/${consulVersion}/${zipName}"
    unzip ${zipName}
    rm ${zipName}
    chmod +x consul
fi
