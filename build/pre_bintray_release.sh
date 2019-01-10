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

set -e

scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ -z "${TRAVIS_TAG}" ]]; then
    echo "not a travis release build, no need to publish it on bintray. Exiting."
    exit 0
fi

TAG_NAME="${TRAVIS_TAG}"
VERSION_NAME="${TAG_NAME#v*}"
RELEASE_DATE="$(git tag -l --format='%(creatordate:short)' "${TAG_NAME}")"

export TAG_NAME VERSION_NAME RELEASE_DATE
envsubst < "${scriptDir}/bintray_release.json.tpl" > "${scriptDir}/bintray_release.json" 

echo "Resulting bintray release spec"
cat "${scriptDir}/bintray_release.json" 
