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

#### Artifactory Variables
artifactory_docker_registry="ystia-yorc-docker-dev-local.jfrog.io"
artifactory_docker_repo="ystia/yorc"

if [[ ! -e ${script_dir}/yorc ]]; then
    cd ${script_dir}
    make
fi

tf_version=$(grep terraform_version ${script_dir}/versions.yaml | awk '{print $2}')
ansible_version=$(grep ansible_version ${script_dir}/versions.yaml | awk '{print $2}')
yorc_version=$(grep yorc_version "${script_dir}/versions.yaml" | awk '{print $2}')
tf_consul_plugin_version=$(grep tf_consul_plugin_version ${script_dir}/versions.yaml | awk '{print $2}')
tf_aws_consul_plugin_version=$(grep tf_aws_consul_plugin_version ${script_dir}/versions.yaml | awk '{print $2}')
tf_openstack_plugin_version=$(grep tf_openstack_plugin_version ${script_dir}/versions.yaml | awk '{print $2}')
tf_google_plugin_version=$(grep tf_google_plugin_version ${script_dir}/versions.yaml | awk '{print $2}')

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
docker build ${BUILD_ARGS} \
        --build-arg BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --build-arg VCS_REF="$(git rev-parse --short HEAD)" \
        --build-arg "TERRAFORM_VERSION=${tf_version}" \
        --build-arg "ANSIBLE_VERSION=${ansible_version}" \
        --build-arg "YORC_VERSION=${yorc_version}" \
        --build-arg "TF_CONSUL_PLUGIN_VERSION=${tf_consul_plugin_version}" \
        --build-arg "TF_AWS_PLUGIN_VERSION=${tf_aws_consul_plugin_version}" \
        --build-arg "TF_OPENSTACK_PLUGIN_VERSION=${tf_openstack_plugin_version}" \
        --build-arg "TF_GOOGLE_PLUGIN_VERSION=${tf_google_plugin_version}" \
        -t "ystia/yorc:${DOCKER_TAG:-latest}" .

if [[ "${TRAVIS}" == "true" ]]; then
    docker save "ystia/yorc:${DOCKER_TAG:-latest}" | gzip > docker-ystia-yorc-${DOCKER_TAG:-latest}.tgz
    ls -lh docker-ystia-yorc-${DOCKER_TAG:-latest}.tgz

    if [[ "${TRAVIS_PULL_REQUEST}" != "false" ]] && [[ -z "${ARTIFACTORY_API_KEY}" ]] ; then
        echo "Building an external pull request, artifactory publication is disabled"
        exit 0
    fi
    
    if [[ -n "${TRAVIS_TAG}" ]] && [[ "${DOCKER_TAG}" != *"-"* ]] ; then
        ## Push Image to the Docker hub
        docker login -u ${DOCKER_HUB_USER} -p ${DOCKER_HUB_PASS}
        docker push "ystia/yorc:${DOCKER_TAG:-latest}"
    else
        ## Push Image on Artifact Docker Registry
        docker tag "ystia/yorc:${DOCKER_TAG:-latest}" "${artifactory_docker_registry}/${artifactory_docker_repo}:${DOCKER_TAG:-latest}"
        curl -fL https://getcli.jfrog.io | sh
        build_name="yorc-travis-ci"
        ./jfrog rt c --user=travis --apikey="${ARTIFACTORY_API_KEY}" --url=https://ystia.jfrog.io/ystia ystia
        ./jfrog rt docker-push --build-name="${build_name}" --build-number="${TRAVIS_BUILD_NUMBER}" "${artifactory_docker_registry}/${artifactory_docker_repo}:${DOCKER_TAG:-latest}" yorc-docker-dev-local
        ./jfrog rt bag "${build_name}" "${TRAVIS_BUILD_NUMBER}" "${script_dir}"
        ./jfrog rt bp "${build_name}" "${TRAVIS_BUILD_NUMBER}"
    fi
fi
