#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

source ${scripts}/logstash_utils.sh

ensure_home_var_is_set

# get LOGSTASH_HOME
source $HOME/.starlings/${NODE}-service.env

log info "Execute change output command with URL : $url"

SEND_SIGHUP="false"
if [[ "${AUTO_RELOAD}" != "true" ]]; then
    # need to reload the config file and restart the pipeline by sending a SIGHUP
    PID=`ps -aux |grep 'logstash/runner.rb' |grep -v 'grep' |awk '{ print $2 }'`
    log info "Got PID $PID"
    SEND_SIGHUP="true"
fi

replace_conf_file $LOGSTASH_HOME/conf/3-1_logstash_outputs.conf "output" $url || error_exit "Reconfiguration failed"

if [[ $SEND_SIGHUP == "true" ]]; then
    log info "No auto-reload, send SIGHUP to $PID"
    kill -1 $PID
fi

log end

