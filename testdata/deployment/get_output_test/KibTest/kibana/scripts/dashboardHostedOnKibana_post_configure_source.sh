#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
CMD_ESDUMP_IMPORT=${scripts}/elasticdump_import.sh

log begin

if [[ ! -f ${dashboard_file} ]]
then
    log info "Dashboard URL is null, so we do nothing"
    exit 0
fi

log info "$0: SOURCE_INSTANCE=${SOURCE_INSTANCE}, TARGET_INSTANCE=${TARGET_INSTANCE}, Dashboard_File=${dashboard_file}"

log debug "${CMD_ESDUMP_IMPORT} ${dashboard_file} called..."
chmod a+x ${CMD_ESDUMP_IMPORT}
${CMD_ESDUMP_IMPORT} ${dashboard_file}

