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


scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
rootDir=$(readlink -f "${scriptDir}/..")

if [[ "${TRAVIS}" != "true" ]] ; then
    echo "This script is designed to publish CI build artifacts"
    exit 0
fi

if [[ "${TRAVIS_PULL_REQUEST}" != "false" ]] && [[ -z "${ARTIFACTORY_API_KEY}" ]] ; then
    echo "Building an external pull request, artifactory publication is disabled"
    exit 0
fi

if [[ -n "${TRAVIS_TAG}" ]] ; then
    deploy_path="yorc-engine-product-ystia-dist/ystia/yorc/dist/${TRAVIS_TAG}/{1}"
elif [[ "${TRAVIS_PULL_REQUEST}" != "false" ]]; then
    deploy_path="yorc-bin-dev-local/ystia/yorc/dist/PR-${TRAVIS_PULL_REQUEST}/{1}"
else
    deploy_path="yorc-bin-dev-local/ystia/yorc/dist/${TRAVIS_BRANCH}/{1}"
fi

curl -fL https://getcli.jfrog.io | sh

build_name="yorc-travis-ci"

./jfrog rt c --apikey="${ARTIFACTORY_API_KEY}" --user=travis --url=https://ystia.jfrog.io/ystia ystia
./jfrog rt u --build-name="${build_name}" --build-number="${TRAVIS_BUILD_NUMBER}" --props="artifactory.licenses=Apache-2.0" --regexp "dist/(yorc-.*.tgz)" "${deploy_path}"
./jfrog rt u --build-name="${build_name}" --build-number="${TRAVIS_BUILD_NUMBER}" --props="artifactory.licenses=Apache-2.0" --regexp "dist/(yorc-server.*-distrib.zip)" "${deploy_path}"
# Do not publish environment variables as it may expose some secrets
#./jfrog rt bce "${build_name}" "${TRAVIS_BUILD_NUMBER}"
./jfrog rt bag "${build_name}" "${TRAVIS_BUILD_NUMBER}" "${rootDir}"
./jfrog rt bp "${build_name}" "${TRAVIS_BUILD_NUMBER}"