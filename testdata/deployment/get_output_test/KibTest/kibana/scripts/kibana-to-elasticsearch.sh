#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin
ensure_home_var_is_set

log debug "$0: Target: ${TARGET_NODE}/${TARGET_INSTANCE} and Source ${SOURCE_NODE}/${SOURCE_INSTANCE}"
log debug "$0: kibanaIp=${kibanaIp} and elasticsearchIp=${elasticsearchIp}"

# If Kibana and Elasticsearch are on same host, Elasticsearch client node is not necessary
[[ "${kibanaIp}" = "elasticsearchIp" ]] && log info "$0: Not needed to install an elasticsearch client node for kibana" && exit 0

#
# Elasticsearch installation
#
source ${scripts}/elasticsearch-service.properties
ES_INSTALL_DIR=${HOME}
ES_HOME_DIR=${ES_INSTALL_DIR}/${ES_UNZIP_FOLDER}
log info "Installing Elasticsearch on ${ES_HOME_DIR}"

echo "ELASTICSEARCH_HOME=${ES_HOME_DIR}" >>${STARLINGS_DIR}/starlings_kibana_env.sh

if [[ -n "${REPOSITORY}" ]] && [[ "${REPOSITORY}" != "DEFAULT" ]] && [[ "${REPOSITORY}" != "null" ]]
then
    ES_DOWNLOAD_PATH="${REPOSITORY}/${ES_ZIP_NAME}"
    ES_HASH_DOWNLOAD_PATH="${ES_DOWNLOAD_PATH}.sha1.txt"
fi

# Elasticsearch installation
( cd ${ES_INSTALL_DIR} && wget ${ES_DOWNLOAD_PATH} ) || error_exit "ERROR: Failed to install Elasticsearch (download problem) !!!"
tar -xzf ${ES_INSTALL_DIR}/${ES_ZIP_NAME} -C ${ES_INSTALL_DIR} || error_exit "ERROR: Failed to install Elasticsearch (untar problem) !!!"

# Configure Elasticsearch as a client node
echo "cluster.name: ${cluster_name}" > ${ES_HOME_DIR}/config/elasticsearch.yml
echo "node.name: client4kibana.${ES_SERVICE_NAME}.service.starlings" >> ${ES_HOME_DIR}/config/elasticsearch.yml
echo "node.master: false" >> ${ES_HOME_DIR}/config/elasticsearch.yml
echo "node.data: false" >> ${ES_HOME_DIR}/config/elasticsearch.yml
echo "network.bind_host: 0.0.0.0" >> ${ES_HOME_DIR}/config/elasticsearch.yml
echo "network.publish_host: ${kibanaIp}" >> ${ES_HOME_DIR}/config/elasticsearch.yml
echo "discovery.zen.ping.unicast.hosts: [\"${ES_SERVICE_NAME}.service.starlings\"]" >> ${ES_HOME_DIR}/config/elasticsearch.yml

log end