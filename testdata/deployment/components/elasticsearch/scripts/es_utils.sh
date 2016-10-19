#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#
#set -x

source ${scripts}/elasticsearch-service.properties

CONFIG_PATH=${HOME}/${ES_UNZIP_FOLDER}/config

comment_existing_property () {
    property_name=$1
    sed -i.bak -e "s/^\s*${property_name}:/#${property_name}:/g" ${CONFIG_PATH}/elasticsearch.yml
}

update_property () {
    property_name=$1
    property_value=$2
    if [[ ! -z "${property_value}" ]] && [[ "${property_value}" != "null" ]]
    then
        if [[ $(grep -c -E "^\s*${property_name}:" ${CONFIG_PATH}/elasticsearch.yml) -gt 1 ]]
        then
            comment_existing_property "${property_name}"
            echo "${property_name}: ${property_value}" >> ${CONFIG_PATH}/elasticsearch.yml
        else
            # replace only the first match
            sed -i -e "0,/^.*${property_name}:.*$/s##${property_name}: ${property_value}#" ${CONFIG_PATH}/elasticsearch.yml
        fi
    fi
}