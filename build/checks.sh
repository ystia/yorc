#!/usr/bin/env bash
# Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#set -x
scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
binDir="${scriptDir}/bin"

export GO111MODULE=on

get_consul_version () {
    grep consul_version "${scriptDir}/../versions.yaml" | awk '{print $2}'
}

error_exit () {
    >&2 echo "${1}"
    if [[ $# -gt 1 ]]
    then
        exit ${2}
    else
        exit 1
    fi
}

install_consul() {
    mkdir -p "${binDir}"
    cd "${binDir}"
    consulVersion=$(get_consul_version)
    zipName="consul_${consulVersion}_$(go env GOHOSTOS)_$(go env GOHOSTARCH).zip"
    wget "https://releases.hashicorp.com/consul/${consulVersion}/${zipName}"
    unzip ${zipName}
    rm ${zipName}
    chmod +x consul
}


if [[ -z "$(which go)" ]]; then
    error_exit "go program should be present in your path"
fi

if [[ ${BUILD_ARGS} == *"-tags"* ]]; then
    error_exit "Variable BUILD_ARGS (\"${BUILD_ARGS}\") contains -tags option, this is deprecated. Use BUILD_TAGS variable to provide a tag list (space-separated) instead."
fi

for tool in $@; do
    #Suppress trailing /... in url if any
    tool="${tool%%/...*}"
    if [[ -z "$(which ${tool##*/})" ]]; then
        error_exit "Tool not found ${tool##*/} doesn't exist. This could be fixed by running 'make tools'"
    fi
done

if [[ ! -x "${binDir}/consul" ]]; then
    rm -f "${binDir}/consul"
    install_consul
else
    installedConsulVersion=$(${binDir}/consul version | grep "Consul v" | cut -d 'v' -f2)
    consulVersion=$(get_consul_version)
    if [[ "${installedConsulVersion}" != "${consulVersion}" ]]; then
        rm -f "${binDir}/consul"
        install_consul
    fi
fi

if [[ ! -r "${HOME}/.ssh/yorc.pem" ]]; then
    echo "Installing a testing ssh key under ${HOME}/.ssh/yorc.pem"
    mkdir -p "${HOME}/.ssh/" || error_exit "Can't create ${HOME}/.ssh/ directory"
    cp "${scriptDir}/yorc.pem"  "${HOME}/.ssh/yorc.pem" || error_exit "Can't copy testing key to ${HOME}/.ssh/ directory"
fi
