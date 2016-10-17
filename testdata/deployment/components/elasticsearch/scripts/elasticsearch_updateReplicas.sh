#!/bin/bash
#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh

log begin

log info "On $(hostname) Updating Elasticsearch with nb_replicas '${nb_replicas}' and index '${index}'"

if [[ "${index}" != "" ]] && [[ "${index}" != "''" ]]; then
    SETTINGS="http://localhost:9200/${index}/_settings"
else
    SETTINGS='http://localhost:9200/_settings'
fi
CURL='/usr/bin/curl'
OP='-XPUT'
PAYLOAD="{\"index\" : {\"number_of_replicas\" : $nb_replicas }}"

log info "Command: $CURL $OP $SETTINGS -d '${PAYLOAD}'"

$CURL $OP $SETTINGS -d "${PAYLOAD}" || error_exit "Failed executing Elasticsearch update with nb_replicas ${nb_replicas} and index ${index}" $?

log end
