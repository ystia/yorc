#!/usr/bin/env bash

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
