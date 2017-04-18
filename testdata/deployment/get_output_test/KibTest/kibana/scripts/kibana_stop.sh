#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

log info "Stopping Kibana..."

sudo systemctl stop kibana.service

#PID=`ps ax |grep 'node/bin/node' |grep -v 'grep' |awk '{ print $1 }'`
#if [ -n "$PID" ]
#then
#  kill -15 $PID
#fi

if [[ -f /tmp/elasticsearch-client-node.pid ]]
then
  log info "Stopping ElasticSearch client node..."
  cat /tmp/elasticsearch-client-node.pid |xargs kill
fi
log end
