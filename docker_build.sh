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


script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ ! -e ${script_dir}/yorc ]]; then
    cd ${script_dir}
    make
fi

tf_version=$(grep terraform_version ${script_dir}/versions.yaml | awk '{print $2}')
ansible_version=$(grep ansible_version ${script_dir}/versions.yaml | awk '{print $2}')

cp ${script_dir}/yorc ${script_dir}/pkg/
cd ${script_dir}/pkg
docker build ${BUILD_ARGS} --build-arg "TERRAFORM_VERSION=${tf_version}" --build-arg "ANSIBLE_VERSION=${ansible_version}" -t "ystia/yorc:${DOCKER_TAG:-latest}" .
