#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#
#set -x


source ${utils_scripts}/utils.sh

log begin

source ${scripts}/es_utils.sh

ensure_home_var_is_set
update_property "path.data" "${path_fs}"

log end