#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015-2016 Bull S.A.S. - All rights reserved
#

. ${utils_scripts}/utils.sh

lock "$(basename $0)"

ensure_home_var_is_set

STARLINGS_DIR="${HOME}/.starlings"
mkdir -p ${STARLINGS_DIR}

if [[ -e "${STARLINGS_DIR}/.${SOURCE_NODE}-preconfiguresourceWanFlag" ]]; then
    unlock "$(basename $0)"
    exit 0
fi

INSTALL_DIR=$(eval readlink -f "${INSTALL_DIR}")

addresses=( $(get_multi_instances_attribute "SERVER_WAN_IP" "TARGET") )
declare -p addresses
# quote array values TODO check if we can use a better  solution to do it
addresses=( ${addresses[@]/#/\"} )
addresses=( ${addresses[@]/%/\"} )
addr_block="$(join_list "," ${addresses[*]})"


cat > ${INSTALL_DIR}/config/4_wan_join.json << EOF
{
  "retry_join_wan": [${addr_block}]
}
EOF

log info "Consul Server configured to connects to server on WAN [${addr_block}]"

touch "${STARLINGS_DIR}/.${SOURCE_NODE}-preconfiguresourceWanFlag"
unlock "$(basename $0)"
