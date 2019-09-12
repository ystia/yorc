#!/usr/bin/env bash
# Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

if [[ -z "${SONAR_TOKEN}" ]] ; then
    echo "No sonar token detected, we are probably building an external PR, lets skip sonar publication..."
    exit 0
fi

cd "${scriptDir}/.."
sed -i -e "s@$(go list)@github.com/ystia/yorc@g" coverage-sonar.out
git fetch --no-tags origin "+refs/heads/develop:refs/remotes/origin/develop" "+refs/heads/release/*:refs/remotes/origin/release/*"
git fetch --unshallow --quiet
sonar-scanner --define "sonar.projectVersion=$(grep "yorc_version" versions.yaml | awk '{print $2}')"
