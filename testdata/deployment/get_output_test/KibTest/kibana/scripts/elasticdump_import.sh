#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

DASHBOARD_FILE=$1

source ${utils_scripts}/utils.sh
log begin
ensure_home_var_is_set
source $HOME/.starlings/starlings_kibana_env.sh

ELASTICSEARCH_URL=http://localhost:9200
KIBANA_INDEX=".kibana"


# Workaround to solve the mapping problem (long instead int) for the fields 'hits' and 'version' to fix the discovery problem
curl -XPUT "${ELASTICSEARCH_URL}/${KIBANA_INDEX}"
curl -XPUT "${ELASTICSEARCH_URL}/${KIBANA_INDEX}/_mapping/search" -d'{"search": {"properties": {"hits": {"type": "integer"}, "version": {"type": "integer"}}}}'
curl -XPUT "${ELASTICSEARCH_URL}/${KIBANA_INDEX}/_mapping/visualization" -d'{"visualization": {"properties": {"hits": {"type": "integer"}, "version": {"type": "integer"}}}}'
curl -XPUT "${ELASTICSEARCH_URL}/${KIBANA_INDEX}/_mapping/dashboard" -d'{"dashboard": {"properties": {"hits": {"type": "integer"}, "version": {"type": "integer"}}}}'


CMD="elasticdump  --input=${DASHBOARD_FILE} --output=${ELASTICSEARCH_URL}/${KIBANA_INDEX} --type=data"
log info "$CMD"
$CMD

