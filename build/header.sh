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
export GO111MODULE=on

if [[ -z "$(which addlicense)" ]] ; then
    >&2 echo "addlicense binary not in path. Should perform 'make tools'."
    exit 1
fi

LICENSE="apache"
OWNER="Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France."
YEAR="2019"

## Add copyright header for go files (test includes)
go list -f '{{range $i, $g := .GoFiles}}{{printf "%s/%s " $.Dir $g}}{{end}} {{range $i, $g := .TestGoFiles}}{{printf "%s/%s " $.Dir $g}}{{end}}' ./... | tr '\n' ' ' | xargs addlicense -c "$OWNER" -l "$LICENSE" -y "$YEAR"

## Add copyright header for bash files
find . -type f -name "*.sh" | grep -v "vendor" | tr "\n" " " | xargs addlicense -c "$OWNER" -l "$LICENSE" -y "$YEAR"