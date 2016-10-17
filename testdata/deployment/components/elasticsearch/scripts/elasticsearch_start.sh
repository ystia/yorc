#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

source ${scripts}/elasticsearch-service.properties

source ${utils_scripts}/utils.sh
ensure_home_var_is_set

source ${scripts}/java_utils.sh
retrieve_java_home "${HOST}"

# Note: Remove -Xmx512m option in JAVA_OPTS added by gigaspaces/tools/groovy/bin/startGroovy
JAVA_OPTS_2=$(echo ${JAVA_OPTS} |sed -s 's|-Xmx[0-9]*[a-zA-Z]||')
export JAVA_OPTS=${JAVA_OPTS_2}

mkdir -p ${HOME}/${ES_UNZIP_FOLDER}/logs
export ES_HEAP_SIZE="${ELASTICSEARCH_HEAP_SIZE}"
nohup ${HOME}/${ES_UNZIP_FOLDER}/bin/elasticsearch > ${HOME}/${ES_UNZIP_FOLDER}/logs/elasticsearch-output.log 2>&1  &
echo "$!" > /tmp/elasticsearch.pid

sleep 3

wait_for_port_to_be_open "127.0.0.1" "9300" 240  || error_exit "Cannot open port 9300"
log end
