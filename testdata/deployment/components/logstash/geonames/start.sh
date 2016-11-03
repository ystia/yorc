#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh

ensure_home_var_is_set

log begin

source ${HOME}/.starlings/${NODE}-service.env

if is_port_open "127.0.0.1" "8500"
then
    # Consul is present: distributed case
    OPTIONS='--noproxy .starlings'
    ES='0.elasticsearch.service.starlings'
else
    # No consul: local case.
    ES='localhost'
fi
log info "Command: curl $OPTIONS -XGET http://${ES}:9200/_cat/indices?v"

log end
