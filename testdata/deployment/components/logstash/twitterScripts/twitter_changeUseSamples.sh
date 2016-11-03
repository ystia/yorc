#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

source ${ls_scripts}/logstash_utils.sh

ensure_home_var_is_set

# get LOGSTASH_HOME
source $HOME/.starlings/${HOST}-service.env

log info "Update Twitter use_samples property: "
log info "    use_samples: ${use_samples}"


SEND_SIGHUP="false"
if [[ "${AUTO_RELOAD}" != "true" ]]; then
    # need to reload the config file and restart the pipeline by sending a SIGHUP
    PID=`ps -aux |grep 'logstash/runner.rb' |grep -v 'grep' |awk '{ print $2 }'`
    log info "Got Logstash PID $PID"
    SEND_SIGHUP="true"
fi

# reconfigure use_samples property
if [[ "$(grep -c "use_samples" "${LOGSTASH_HOME}/conf/1-${NODE}_logstash_inputs.conf")" != "0" ]]; then
    replace_conf_value $LOGSTASH_HOME/conf/1-${NODE}_logstash_inputs.conf "use_samples" $use_samples || error_exit "Reconfiguration failed"
else
    add_conf_property $LOGSTASH_HOME/conf/1-${NODE}_logstash_inputs.conf "use_samples" $use_samples || error_exit "Reconfiguration failed"
fi

if [[ $SEND_SIGHUP == "true" ]]; then
    log info "No auto-reload, send SIGHUP to $PID"
    kill -1 $PID
fi

log end