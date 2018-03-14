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
#set -e

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

#### Bintray Variables
bintray_api_user="stebenoist"
bintray_docker_registry="ystia-docker-yorc.bintray.io"
bintray_docker_repo="ystia/yorc"

if [[ ! -e ${script_dir}/yorc ]]; then
    cd ${script_dir}
    make
fi

tf_version=$(grep terraform_version ${script_dir}/versions.yaml | awk '{print $2}')
ansible_version=$(grep ansible_version ${script_dir}/versions.yaml | awk '{print $2}')

if [[ "${TRAVIS}" == "true" ]]; then
    if [[ "${TRAVIS_PULL_REQUEST}" == "false" ]] ; then
        if [[ -n "${TRAVIS_TAG}" ]] ; then
            DOCKER_TAG="$(echo "${TRAVIS_TAG}" | sed -e 's/^v\(.*\)$/\1/')"
        else
            case ${TRAVIS_BRANCH} in
            develop) 
                DOCKER_TAG="latest";;
            *) 
                # Do not build a container for other branches
                echo "No container is built for other branches than develop."
                exit 0;;
            esac
        fi
    else 
        DOCKER_TAG="PR-${TRAVIS_PULL_REQUEST}"
    fi
fi

cp ${script_dir}/yorc ${script_dir}/pkg/
cd ${script_dir}/pkg
docker build ${BUILD_ARGS} --build-arg "TERRAFORM_VERSION=${tf_version}" --build-arg "ANSIBLE_VERSION=${ansible_version}" -t "ystia/yorc:${DOCKER_TAG:-latest}" .

if [[ "${TRAVIS}" == "true" ]]; then
    docker save "ystia/yorc:${DOCKER_TAG:-latest}" | gzip > docker-ystia-yorc-${DOCKER_TAG:-latest}.tgz
    ls -lh docker-ystia-yorc-${DOCKER_TAG:-latest}.tgz

    if [[ -n "${TRAVIS_TAG}" ]] && [[ "${DOCKER_TAG}" != *"-"* ]] ; then
        ## Push Image to the Docker hub
        docker login -u ${DOCKER_HUB_USER} -p ${DOCKER_HUB_PASS}
        docker push "ystia/yorc:${DOCKER_TAG:-latest}"
    else
        ## Push Image on Bintray Docker Registry
        docker login -u ${bintray_api_user} -p $BINTRAY_API_KEY ${bintray_docker_registry}
        docker tag "ystia/yorc:${DOCKER_TAG:-latest}" ${bintray_docker_registry}/${bintray_docker_repo}:${DOCKER_TAG:-latest}
        docker push ${bintray_docker_registry}/${bintray_docker_repo}:${DOCKER_TAG:-latest}
    fi
fi
