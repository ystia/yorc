#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#
# Collection of utilities in order to interact with consul

#################################################################################
# Get a new unique positive integer (>= 0) representing a identifier using
# Consul's Key/Value store.
#
# Note: this function relies on the jq utility that should be installed.
#
# Globals:
#   CONSUL_URI: Optional, defaults to 'http://127.0.0.1:8500'
# Arguments:
#   1/ Path in key value store (ie 'service/my-service' will create the following
#      key '${CONSUL_URI}/v1/kv/service/my-service/nextId')
#   2/ Behavior in case of update conflict (optional):
#       - "exit": exits with return code 1 (default for backward compatibility)
#       - "return": returns with return code 1
#       - "retry": keep trying to get an id after a delay
# Returns:
#   The generated id
#################################################################################
get_new_id () {
    local service_path="$1"
    local conflict_behavior="exit"
    if [[ $# -gt 1 ]] ; then
        conflict_behavior="$2"
    fi
    local consul_uri="http://127.0.0.1:8500"
    if [[ -n "${CONSUL_URI}" ]] ; then
        consul_uri="${CONSUL_URI}"
    fi
    is_done="false"
    while [[ "${is_done}" == "false" ]]; do
        # Get current value if any and modify index for atomic update
        result=$(curl -s -X GET "${consul_uri}/v1/kv/${service_path}/nextId")
        if [[ -z "${result}" ]] ; then
            id=0
            index=0
        else
            read id index <<< $(echo "${result}" | jq -r ".[]|.Value,.ModifyIndex")
            if [[ "${id}" == "null" ]]; then
                id=0
            else
                # value is base64 encoded
                id=$(echo ${id} | base64 --decode)
            fi
        fi
        next_id=$((${id} + 1))
        # Perform the atomic update
        if [[ "true" == "$(curl -s -X PUT -d "${next_id}" "${consul_uri}/v1/kv/${service_path}/nextId?cas=${index}")" ]]; then
            is_done="true"
        fi
        if [[ "${is_done}" == "false" ]]; then
            case "${conflict_behavior}" in
                "return")
                    >&2 echo "WARNING: Can't acquire our unique id from consul."
                    return 1
                    ;;
                "retry")
                    >&2 echo "WARNING: Can't acquire our unique id from consul. Let's retry after a short delay."
                    sleep $(( ( RANDOM % 5 )  + 1 ))
                    ;;
                *)
                    >&2 echo "WARNING: Can't acquire our unique id from consul. Exiting..."
                    exit 1
                    ;;
            esac
        fi
    done
    echo ${id}
}
