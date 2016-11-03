#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

ensure_home_var_is_set

if isServiceConfigured; then
    log end "Logstash component '${NODE}' already configured"
    exit 0
fi

# get LOGSTASH_HOME
source $HOME/.starlings/${NODE}-service.env


mkdir -p "${LOGSTASH_HOME}/conf"
mkdir -p "${LOGSTASH_HOME}/logs"
mkdir -p "${LOGSTASH_HOME}/lib/logstash/certs"
mkdir -p "${LOGSTASH_HOME}/patterns"

if [[ "${STDOUT}" = "true" ]]
then
    # A new output config file is added in case STDOUT is true.
     log info "Configure stdout for logstash"
     cp ${conf}/3-stdout_logstash_outputs.conf $LOGSTASH_HOME/conf
else
     log info "Configure logstash without stdout"
     rm -f $LOGSTASH_HOME/conf/3-stdout_logstash_outputs.conf
fi

log info "Copy configuration files and certificates..."

cp "${inputs_conf}" "${LOGSTASH_HOME}/conf/1-1_logstash_inputs.conf"
cp "${filters_conf}" "${LOGSTASH_HOME}/conf/2-1_logstash_filters.conf"
cp "${outputs_conf}" "${LOGSTASH_HOME}/conf/3-1_logstash_outputs.conf"

cp "${scripts}/java.security" "${LOGSTASH_HOME}/java.security"
cp "${scripts}/logback" "${LOGSTASH_HOME}/patterns/logback"

cp ${certificates}/* "${LOGSTASH_HOME}/lib/logstash/certs"
cp "${private_key}" "${LOGSTASH_HOME}/lib/logstash/certs/default-logstash-forwarder.key"
cp "${certificate}" "${LOGSTASH_HOME}/lib/logstash/certs/default-logstash-forwarder.crt"

sed -i -e "s@#LOGSTASH_CERTIFICATES_DIR#@${LOGSTASH_HOME}/lib/logstash/certs@g" ${LOGSTASH_HOME}/conf/*

# For multiple level LKEK purpose only, add /etc/hosts with name's of kafka n+1 level find into artifacts extra_host
#sudo bash -c "cat ${extra_host} >>/etc/hosts"
#echo `date` ">>>> 'extra_hosts' added in '/etc/hosts'"

setServiceConfigured

log end
