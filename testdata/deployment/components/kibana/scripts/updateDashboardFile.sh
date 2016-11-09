#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh

log begin

# Check the dashboard_url parameter value validity
[[ -z "${dashboard_url}" ]] && error_exit "ERROR: dashboard_url parameter is required !"

log info "Update the dashboard with ${dashboard_url} ..."

CMD_IMPORT_DB=${scripts}/importDashboardOnKibana.sh
log debug "${CMD_IMPORT_DB} with url=${dashboard_url}"
export url=${dashboard_url}
bash ${CMD_IMPORT_DB}

