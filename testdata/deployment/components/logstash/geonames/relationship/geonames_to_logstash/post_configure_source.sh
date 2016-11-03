#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
source ${scripts}/logstash_utils.sh

ensure_home_var_is_set
log begin

log info "TARGET=${TARGET_NODE} SOURCE=${SOURCE_NODE}"

source ${HOME}/.starlings/${TARGET_NODE}-service.env
source ${HOME}/.starlings/${SOURCE_NODE}-service.env

log info "Remove default es output plugin"
rm $LOGSTASH_HOME/conf/*elasticsearch*.conf

log info "Copy geonames_parse.conf file to $LOGSTASH_HOME/conf"
cp ${conf}/geonames_parse.conf $LOGSTASH_HOME/conf

# Update GeoNames config file
if is_port_open "127.0.0.1" "8500"
then
    es_host_name="elasticsearch.service.starlings"
else
    es_host_name="localhost"
fi
es_port="9200"
ES_HOST=$es_host_name:$es_port

sed -i -e "s@#GEONAMES_HOME#@${GEONAMES_HOME}@g" ${LOGSTASH_HOME}/conf/geonames_parse.conf
sed -i -e "s@#FNAME#@${FNAME}@g" ${LOGSTASH_HOME}/conf/geonames_parse.conf
sed -i -e "s@#ES_HOST#@${ES_HOST}@g" ${LOGSTASH_HOME}/conf/geonames_parse.conf
sed -i -e "s@#INDEX#@${INDEX}@g" ${LOGSTASH_HOME}/conf/geonames_parse.conf

reload_configuration $AUTO_RELOAD

# Retain some LOGSTASH properties necessary for GeoNames (re-)configure
echo "AUTO_RELOAD=$AUTO_RELOAD" >> ${HOME}/.starlings/${SOURCE_NODE}-service.env
echo "LOGSTASH_HOME=$LOGSTASH_HOME" >> ${HOME}/.starlings/${SOURCE_NODE}-service.env

log end
