#!/usr/bin/env bash

set -euo pipefail

comp_date=$(date --date="${FROM_DATE:=14 days ago}"  +"%Y-%m-%dT%H:%M:%S.000Z")

local_dist_path="yorc-bin-dev-local/ystia/yorc/dist"

all_paths=$(jfrog rt s "${local_dist_path}/*/yorc-*.tgz" --limit 0 | jq -r ".[]| [.modified, .path] | @tsv" | sed -e "s@${local_dist_path}/\(.*\)/yorc-.*\.tgz@\1@g" | sort)

function get_pr_state() {
    gh pr view "${1}" --json state | jq -r ".state"
}

function does_branch_exit() {
    gh api --silent "/repos/:owner/:repo/branches/${1}" 2> /dev/null
    return $?
}

function delete_artifactory_path() {
    jfrog rt del --quiet "${local_dist_path}/${1}" || echo "failed to delete ${local_dist_path}/${1}"
}

echo "${all_paths}" | while read line ; do
    item_date=$(echo "$line" | awk '{print $1}')
    if [[ "${item_date}" > "${comp_date}" ]] ; then
        continue
    fi
    ref=$(echo "${line}" | awk -F '\t' '{print $2}')
    if [[ "${ref}" == PR-* ]] ; then
        if [[ "$(get_pr_state "${ref##*PR-}")" != "OPEN" ]] ; then
            delete_artifactory_path "${ref}"
        fi
    else
        if ! does_branch_exit "${ref}" ; then
            delete_artifactory_path "${ref}"
        fi
    fi
done
