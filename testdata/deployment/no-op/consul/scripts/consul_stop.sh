#!/usr/bin/env bash

INSTALL_DIR=$(eval readlink -f "${INSTALL_DIR}")
. ${utils_scripts}/utils.sh

log info "Gracefully leaving consul cluster"
${INSTALL_DIR}/consul leave
exit $?
