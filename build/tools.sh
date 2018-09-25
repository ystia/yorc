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

if [[ -z "$(which go)" ]]; then
    error_exit "go program should be present in your path"
fi

if [[ -z "$GOPATH" ]]; then
    export GOPATH="$(go env GOPATH)"
fi

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