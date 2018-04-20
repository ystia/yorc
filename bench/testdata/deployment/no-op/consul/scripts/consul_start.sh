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


INSTALL_DIR=$(eval readlink -f "${INSTALL_DIR}")

. ${utils_scripts}/utils.sh
log begin

# Setup GOMAXPROCS to the number of cpu
export GOMAXPROCS=$(cat /proc/cpuinfo | grep processor | wc -l)

nohup ${INSTALL_DIR}/consul agent -pid-file="${INSTALL_DIR}/work/consul.pid" -config-dir="${INSTALL_DIR}/config" > ${INSTALL_DIR}/logs/consul.log 2>&1 </dev/null &
timeout=$((600))
time=$((0))
while [[ ! -e  ${INSTALL_DIR}/work/consul.pid ]]; do
  sleep 1
  time=$((time + 1))
  [[ ${time} -gt ${timeout} ]] && { echo "Failed to start consul!!!"; exit 1; }
done
log end "Consul started."

