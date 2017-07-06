#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh

log begin

source ${scripts}/java_utils.sh

ensure_home_var_is_set
source ${HOME}/.starlings/starlings_kibana_env.sh

# Export JAVA_HOME
retrieve_java_home "${HOST}"

if [[ ! -z "${ELASTICSEARCH_HOME}" ]]
then
    # ElasticSearch Client Node may running in case of relaunching process after unexpected process death,
    # Stop it before
    if is_port_open "127.0.0.1" "9200"
    then
        if [[ -f /tmp/elasticsearch-client-node.pid ]]
        then
          log info ">>>> Stopping ElasticSearch client node"
          cat /tmp/elasticsearch-client-node.pid |xargs kill
        fi
    fi

    log info "Starting ElasticSearch Client Node"
    # Note: Remove -Xmx512m option in JAVA_OPTS added by gigaspaces/tools/groovy/bin/startGroovy
    JAVA_OPTS_2=$(echo ${JAVA_OPTS} |sed -s 's|-Xmx[0-9]*[a-zA-Z]||')
    export JAVA_OPTS=${JAVA_OPTS_2}
    export ES_HEAP_SIZE="${ELASTICSEARCH_HEAP_SIZE}"

    mkdir -p ${ELASTICSEARCH_HOME}/logs
    nohup ${ELASTICSEARCH_HOME}/bin/elasticsearch -p /tmp/elasticsearch-client-node.pid >${ELASTICSEARCH_HOME}/logs/elasticsearch-output.log 2>&1 &
    sleep 3
    wait_for_port_to_be_open 127.0.0.1 9200 120 10 || error_exit "Unable to start the ElasticSearch Client Node"
fi


# Kibana may running in case of relaunching process after unexpected process death,
# Stop it before
sudo systemctl stop kibana.service


log info "Starting Kibana..."
sudo systemctl start kibana.service
sleep 3
wait_for_port_to_be_open "127.0.0.1" 5601 || error_exit "Cannot open port 5601"

log end
