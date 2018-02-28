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
cd ${scriptDir}/..
export PATH=${scriptDir}:$PATH

pkgs=$(go list ./...)
pkgsList=$(echo "${pkgs}" | tr '\n' ',' | sed -e 's@^\(.*\),$@\1@')
profDir="${scriptDir}/coverprofiles"
rm -rf ${profDir}

echo "mode: atomic" > coverage.txt
for d in ${pkgs}; do
    pOutName="${profDir}/profile-$(echo "$d" | tr '/' '-').out"
    go test -coverprofile=${pOutName} -covermode=atomic -coverpkg="${pkgsList}" $d 2>&1 | grep -v "warning: no packages being tested depend on"
done

gocovermerge -output coverage.txt ${profDir}/*.out
