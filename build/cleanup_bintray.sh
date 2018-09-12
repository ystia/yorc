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


BINTRAY_USER="stebenoist"

date_lang="1 month ago"
purge_date="$(date --date="${date_lang}" --iso-8601=seconds)"

if [[ -z "${BINTRAY_API_KEY}" ]]; then
	echo "please setup BINTRAY_API_KEY variable";
	exit 1
fi

echo "cleanup docker PRs older than ${date_lang}"

for tag in $(curl -s -u "${BINTRAY_USER}:${BINTRAY_API_KEY}" "https://api.bintray.com/packages/ystia/yorc/ystia:yorc/files" | jq ".[] | {version: .version,created: .created} | select(.created < \"${purge_date}\" and (.version | startswith(\"PR-\")))| .version" -r); do
	echo "deleting tag ${tag}"
	curl -s -u "${BINTRAY_USER}:${BINTRAY_API_KEY}" "https://api.bintray.com/packages/ystia/yorc/ystia:yorc/versions/${tag}" -X DELETE | jq '.message' -r
	echo
done


echo "cleanup engine snapshots older than ${date_lang}"

for p in $(curl -s -u "${BINTRAY_USER}:${BINTRAY_API_KEY}" "https://api.bintray.com/packages/ystia/yorc-engine/distributions/files" | jq ".[] | select((.path | startswith(\"snapshots\")) and ( .path | startswith(\"snapshots/develop\") | not ) and (.created < \"${purge_date}\" )) | .path" -r) ; do
	echo "deleting path $p"
	curl -s -u "${BINTRAY_USER}:${BINTRAY_API_KEY}" "https://api.bintray.com/content/ystia/yorc-engine/${p}" -X DELETE | jq '.message' -r
	echo
done
