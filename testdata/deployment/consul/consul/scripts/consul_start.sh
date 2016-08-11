#!/usr/bin/env bash

INSTALL_DIR=$(eval readlink -f "${INSTALL_DIR}")
. ${utils_scripts}/utils.sh

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
log info "Consul started."
exit 0

