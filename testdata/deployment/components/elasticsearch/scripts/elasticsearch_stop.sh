#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

source ${utils_scripts}/utils.sh
log begin

cat /tmp/elasticsearch.pid |xargs kill

log end