#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

ensure_home_var_is_set

if isServiceInstalled; then
    log end "Logstash component '${NODE}' already installed"
    exit 0
fi

log info "Environment variables : source is ${scripts} - home is ${HOME}"

# source the service properties
. ${scripts}/logstash-service.properties

LS_HOME=$HOME/${LS_UNZIP_FOLDER}

if [[ -n "${REPOSITORY}" ]] && [[ "${REPOSITORY}" != "DEFAULT" ]] && [[ "${REPOSITORY}" != "null" ]]; then
    LS_DOWNLOAD_PATH="${REPOSITORY}/${LS_ZIP_NAME}"
    LS_HASH_DOWNLOAD_PATH="${LS_DOWNLOAD_PATH}.sha1.txt"
fi

log info "Downloading logstash ${LS_VERSION} from ${LS_DOWNLOAD_PATH} ..."
wget  -O $HOME/${LS_ZIP_NAME} "${LS_DOWNLOAD_PATH}"
log info "Downloaded to $HOME/${LS_ZIP_NAME}"

# TODO ckecksum
#log info "A4C11 - Downloading checksum files ..."
#https://download.elastic.co/logstash/logstash/logstash-2.1.0.tar.gz.sha1.txt
#wget -nc -P $HOME $hashDownloadPath

log info "Extract $HOME/${LS_ZIP_NAME} to ${HOME}"
tar xzf $HOME/${LS_ZIP_NAME} -C ${HOME}

#chmod $LS_HOME/bin +x excludes: "*.bat,*.exe"
cd ${LS_HOME}/bin
find -type f -name "*.sh" -exec chmod +x \{\} \;


echo "LOGSTASH_HOME=$LS_HOME" > ${HOME}/.starlings/${NODE}-service.env

setServiceInstalled

log end