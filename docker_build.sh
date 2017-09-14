#!/usr/bin/env bash

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ ! -e ${script_dir}/janus ]]; then
    cd ${script_dir}
    make
fi

tf_version=$(grep terraform_version ${script_dir}/versions.yaml | awk '{print $2}')
ansible_version=$(grep ansible_version ${script_dir}/versions.yaml | awk '{print $2}')

cp ${script_dir}/janus ${script_dir}/pkg/
cd ${script_dir}/pkg
docker build ${BUILD_ARGS} --build-arg "TERRAFORM_VERSION=${tf_version}" --build-arg "ANSIBLE_VERSION=${ansible_version}" -t "starlings/janus:${DOCKER_TAG:-latest}" .
