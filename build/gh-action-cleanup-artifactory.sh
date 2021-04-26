#!/usr/bin/env bash

set -euo pipefail

comp_date=$(date --date="${FROM_DATE:=14 days ago}"  +"%Y-%m-%dT%H:%M:%S.000Z")

local_bin_dist_path="yorc-bin-dev-local/ystia/yorc/dist"
local_docker_path="yorc-docker-dev-local/ystia/yorc"

bin_paths=$(jfrog rt s "${local_bin_dist_path}/*/yorc-*.tgz" --limit 0 | jq -r ".[]| [.modified, .path] | @tsv" | sed -e "s@\(${local_bin_dist_path}/\(.*\)\)/yorc-.*\.tgz@\2\t\1@g")
docker_paths=$(jfrog rt s "${local_docker_path}/PR-*/manifest.json" --limit 0 | jq -r ".[]| [.modified, .path] | @tsv" | sed -e "s@\(${local_docker_path}/\(PR-.*\)\)/manifest.json@\2\t\1@g")

all_paths=$(echo -e "${bin_paths}\n${docker_paths}" | sort)

function get_pr_state() {
    gh pr view "${1}" --json state | jq -r ".state"
}

function does_branch_exit() {
    gh api --silent "/repos/:owner/:repo/branches/${1}" 2> /dev/null
    return $?
}

function delete_artifactory_path() {
    jfrog rt del --quiet "${1}" || echo "failed to delete ${1}"
}

echo "${all_paths}" | while read line ; do
    item_date=$(echo "$line" | awk '{print $1}')
    if [[ "${item_date}" > "${comp_date}" ]] ; then
        continue
    fi
    ref=$(echo "${line}" | awk -F '\t' '{print $2}')
    artifact_path=$(echo "${line}" | awk -F '\t' '{print $3}')
    if [[ "${ref}" == PR-* ]] ; then
        if [[ "$(get_pr_state "${ref##*PR-}")" != "OPEN" ]] ; then
            delete_artifactory_path "${artifact_path}"
        fi
    else
        if ! does_branch_exit "${ref}" ; then
            delete_artifactory_path "${artifact_path}"
        fi
    fi
done
