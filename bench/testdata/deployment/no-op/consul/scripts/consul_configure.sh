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


. ${utils_scripts}/utils.sh
INSTALL_DIR=$(eval readlink -f "${INSTALL_DIR}")
log begin

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
    log end "Consul configured to run in server mode"
else
    # In agent mode the preconfigure_source script already setup the connection to masters
    log end "Consul configured to run in agent mode"
fi

cat > ${INSTALL_DIR}/config/2_datacenter.json << EOF
{
  "datacenter": "${DATACENTER}"
}
EOF
