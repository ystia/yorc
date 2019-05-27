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


set -eo pipefail

#set -x

function help () {
    echo "$0 [-e <Yorc_Engine_PR_Number>] [-p <Plugin_PR_number>] [-v <values_file>] [-y]

        bootstraps Yorc with binaries of the given Pull Request number
        an optional values file could be given with the -v flag to
        configure the bootstrap process.

        -e: Yorc Engine PR number, defaults to develop if not provided

        -p: Yorc a4c Plugin PR number, defaults to develop if not provided

        -v: a Yorc bootstrap values file (see https://yorc.readthedocs.io/en/latest/bootstrap.html#bootstrapping-the-setup-using-a-configuration-file)

        -y: assume yes to confirmations (useful for batch scripts)
" >&2
}

CONFIRM=""

function confirmOrExit ()  {
    msg="$1"

    if [[ -n "${CONFIRM}" ]] ; then
        return
    fi

    while true ; do
        read -r -p "${msg}? (Y/n)" confirm
        case "${confirm}" in
            "" | "y" | "Y")
                return
                ;;
            "n" | "N")
                exit 0
                ;;
        esac
    done
}


function getURLFromPart () {
    file=$(curl -s "$1/" |  grep -o -P "$2" | tail -1 ) || return
    if [[ -z "$file" ]] ; then
        return
    fi
    echo "$1/$file"
}

function getYorcURLFromPart () {
    getURLFromPart "$1" 'yorc-[0-9].*?.tgz'
}

function getYorcPluginURLFromPart () {
    getURLFromPart "$1" 'alien4cloud-yorc-plugin-[0-9].*?.zip'
}

while getopts ":yv:e:p:" opt; do
  case $opt in
    v)
      VALUES_FILE=(--values "${OPTARG}")
      ;;
    p)
      PLUGIN_PR=${OPTARG}
      ;;
    e)
      ENGINE_PR=${OPTARG}
      ;;
    y)
      CONFIRM="Y"
      ;;
    \?)
      help
      echo "Invalid option: -$OPTARG" >&2
      exit 1
      ;;
    :)
      help
      echo "Option -$OPTARG requires an argument." >&2
      exit 1
      ;;
  esac
done

mkdir -p work

if [[ -n "${ENGINE_PR}" ]] ; then
    YORC_DOWNLOAD_URL=$(getYorcURLFromPart "https://ystia.jfrog.io/ystia/yorc-bin-dev-local/ystia/yorc/dist/PR-${ENGINE_PR}")
fi

if [[ -z "${YORC_DOWNLOAD_URL}" ]] ; then
    confirmOrExit "PR number not provided or not found on Artifactory for Yorc engine, would you like to use develop"
    YORC_DOWNLOAD_URL=$(getYorcURLFromPart "https://ystia.jfrog.io/ystia/yorc-bin-dev-local/ystia/yorc/dist/develop")
fi

export YORC_DOWNLOAD_URL


if [[ -n "${PLUGIN_PR}" ]] ; then
    YORC_PLUGIN_DOWNLOAD_URL=$(getYorcPluginURLFromPart "https://ystia.jfrog.io/ystia/yorc-a4c-plugin-bin-dev-local/ystia/yorc-a4c-plugin/dist/PR-${PLUGIN_PR}")
fi

if [[ -z "${YORC_PLUGIN_DOWNLOAD_URL}" ]] ; then
    confirmOrExit "PR number not provided or not found on on Artifactory for Yorc A4C Plugin, would you like to use develop"
    YORC_PLUGIN_DOWNLOAD_URL=$(getYorcPluginURLFromPart "https://ystia.jfrog.io/ystia/yorc-a4c-plugin-bin-dev-local/ystia/yorc-a4c-plugin/dist/develop")
fi

export YORC_PLUGIN_DOWNLOAD_URL

echo "Downloading ${YORC_DOWNLOAD_URL} please be patient..."
curl "${YORC_DOWNLOAD_URL}" -o work/yorc.tgz

tar xzvf work/yorc.tgz -C work

./work/yorc bootstrap "${VALUES_FILE[@]}" --yorc_download_url "${YORC_DOWNLOAD_URL}" --yorc_plugin_download_url "${YORC_PLUGIN_DOWNLOAD_URL}"
