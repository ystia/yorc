#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#
source ${utils_scripts}/utils.sh
log begin

ensure_home_var_is_set

lock "$(basename $0)"

# get LOGSTASH_HOME
source $HOME/.starlings/${SOURCE_NODE}-service.env

if [[ -e "${STARLINGS_DIR}/.${SOURCE_NODE}-ls2esFlag" ]]; then
    log info "Logstash component '${SOURCE_NODE}' already configured with Elasticsearch output"
    unlock "$(basename $0)"
    exit 0
fi


log info "Connecting Logstash to Elasticsearch"
log info "Environment variables : source is ${scripts} - home is ${HOME}"
log info "Elasticsearch cluster name is ${cluster_name}"

# Create conf directory if not already created
mkdir -p ${LOGSTASH_HOME}/conf                                                                                  /
touch "${LOGSTASH_HOME}/conf/3-elasticsearch_logstash_outputs.conf"
# Take into account elk-all-in-one topo. Is consul agent present ?
IS_CONSUL=1
if is_port_open "127.0.0.1" "8500"
then
    host_name="elasticsearch.service.starlings"
else
    host_name="localhost"
fi
port="9200"
host=$host_name:$port
log info "Elasticsearch host is $host"

#echo -e "output {\n\telasticsearch {\n\t\tcluster => \"${cluster_name}\"\n\t\tprotocol => node\n\t}\n}" >>${LOGSTASH_HOME}/conf/3-elasticsearch_logstash_outputs.conf
echo -e "output {\n\telasticsearch {\n\t\thosts => [\"$host\"] }\n}" >>${LOGSTASH_HOME}/conf/3-elasticsearch_logstash_outputs.conf
log info "A4C configure elasticsearch cluster ${cluster_name}"

touch "${STARLINGS_DIR}/.${SOURCE_NODE}-ls2esFlag"
log info "Logstash connected to Elasticsearch host ${host}"

unlock "$(basename $0)"

log end