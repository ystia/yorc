#!/usr/bin/env bash

. ${utils_scripts}/utils.sh

INSTALL_DIR=$(eval readlink -f "${INSTALL_DIR}")


if [[ ! -e ${INSTALL_DIR}/work/.agentmode ]]; then
    log info "Configuring Consul to run in server mode"
    # We are not configured to connect to a server so let's start as a server by our own
    addresses=( $(get_multi_instances_attribute "IP_ADDRESS") )
    number_of_masters=${#addresses[@]}
    # quote array values TODO check if we can use a better  solution to do it
    addresses=( ${addresses[@]/#/\"} )
    addresses=( ${addresses[@]/%/\"} )
    addr_block="$(join_list "," ${addresses[*]})"
    sed -i -e "/client_addr/ a\  \"server\": true,\n\
  \"bootstrap_expect\": ${number_of_masters},\n\
  \"retry_join\": [${addr_block}]," ${INSTALL_DIR}/config/1_main_conf.json
  if [[ -n "${WAN_ADDRESS}" ]]; then
cat > ${INSTALL_DIR}/config/3_wan_address.json << EOF
{
  "advertise_addr_wan": "${WAN_ADDRESS}"
}
EOF
  fi
else
    # In agent mode the preconfigure_source script already setup the connection to masters
    log info "Consul configured to run in agent mode"
fi

cat > ${INSTALL_DIR}/config/2_datacenter.json << EOF
{
  "datacenter": "${DATACENTER}"
}
EOF
