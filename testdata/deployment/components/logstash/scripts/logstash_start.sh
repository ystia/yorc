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

source ${scripts}/java_utils.sh

retrieve_java_home "${HOST}"

# overload JAVACMD variable (used in the logstash scripts)
export JAVACMD="${JAVA_HOME}/bin/java"
unset CLASSPATH

# Note: java.security file cannot be under $LOGSTASH_HOME/conf
export LS_JAVA_OPTS="-Djava.security.properties=${LOGSTASH_HOME}/java.security"
# Note: LS_HEAP_SIZE is only for max size
export LS_HEAP_SIZE="${LOGSTASH_HEAP_SIZE}"
export LS_JAVA_OPTS="${LS_JAVA_OPTS} -Xms${LOGSTASH_HEAP_SIZE}"
# Note: use LS_JAVA_OPTS instead JAVA_OPTS to not override Logstash java default options
export LS_JAVA_OPTS="${LS_JAVA_OPTS} ${JAVA_OPTS}"
unset JAVA_OPTS


JAVA_OPTS_2=$(echo ${JAVA_OPTS} |sed -s 's|-Xmx[0-9]*[a-zA-Z]||')
export JAVA_OPTS=${JAVA_OPTS_2}

# Start logstash if not already running
PID=`ps -aux |grep 'logstash/runner.rb' |grep -v 'grep' |awk '{ print $2 }'`
if [ -n "$PID" ]; then
    log end "logstash already started"
    exit 0
fi

case "${LOGSTASH_LOG_LEVEL}" in
    "debug")
        log_level_option="--debug"
        ;;
    "verbose")
        log_level_option="--verbose"
        ;;
    "debug")
        log_level_option="--debug"
        ;;
    *)
        log_level_option=" "
        ;;
esac

if [[ "${AUTO_RELOAD}" == "true" ]]; then
     log info "Logstash configuration : auto_reload is set and reload_interval is ${RELOAD_INTERVAL}"
     start_reload ${LOGSTASH_HOME} ${RELOAD_INTERVAL} ${STDOUT} $log_level_option || error_exit "Logstash start failed"
else
     log info "Logstash configuration : auto_reload is not set"
     start ${LOGSTASH_HOME} ${STDOUT} $log_level_option || error_exit "Logstash start failed"
fi

log end


