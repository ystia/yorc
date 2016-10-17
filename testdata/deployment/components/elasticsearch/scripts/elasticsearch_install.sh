#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#
#set -x

source ${utils_scripts}/utils.sh

log begin

# To set variables for the proxy
ensure_home_var_is_set

source ${scripts}/elasticsearch-service.properties

if isServiceInstalled; then
    log end "Elasticsearch component '${NODE}' already installed"
    exit 0
fi

# Install dependencies
#   - jq for json parsing
bash ${utils_scripts}/install-components.sh jq

if [[ -n "${REPOSITORY}" ]] && [[ "${REPOSITORY}" != "DEFAULT" ]] && [[ "${REPOSITORY}" != "null" ]]; then
    ES_DOWNLOAD_PATH="${REPOSITORY}/${ES_ZIP_NAME}"
    ES_HASH_DOWNLOAD_PATH="${ES_DOWNLOAD_PATH}.sha1"
fi

#Elasticsearch installation
wget ${ES_DOWNLOAD_PATH} -O ${HOME}/${ES_ZIP_NAME} || error_exit "ERROR: Failed to install Elasticsearch (download problem) !!!"
tar xzf ${HOME}/${ES_ZIP_NAME} -C ${HOME} || error_exit "ERROR: Failed to install Elasticsearch (untar problem) !!!"
# Checsum control
wget ${ES_HASH_DOWNLOAD_PATH} -O ${HOME}/${ES_ZIP_NAME}.sha1
(cd ${HOME}; echo "$(cat ${ES_ZIP_NAME}.sha1) ${ES_ZIP_NAME}" | sha1sum -c) || error_exit "ERROR: Checksum validation failed for downloaded Elasticsearch binary"

bash ${scripts}/curator_install.sh

# Increase the maximum file open limit to avoid a warning at elasticsearch startup
sudo bash -c 'echo "* - nofile 65536" >>/etc/security/limits.conf'

setServiceInstalled
log end "Elasticsearch installed at ${HOME}"
