#!/usr/bin/env bash

set -e

source ${utils_scripts}/utils.sh

bash ${utils_scripts}/install-components.sh unzip

log info "Installing beats dashboards from ${dashboards_zip}"

tmpdir=$(mktemp -d)

unzip ${dashboards_zip} -d ${tmpdir}

cd $(dirname $(find ${tmpdir} -name load.sh))

log info "Loading beats dashboards..."

./load.sh

log info "Beats dashboards loaded."
