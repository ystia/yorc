#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#
#set -x

#
# TODO ? Is the service name must also include the cluster name ?
#

source ${utils_scripts}/utils.sh

log begin

source ${scripts}/elasticsearch-service.properties

source ${consul_utils}/utils.sh
source ${scripts}/es_utils.sh

ensure_home_var_is_set

source ${scripts}/java_utils.sh
retrieve_java_home "${HOST}"

if isServiceConfigured; then
     log end "Elasticsearch component '${NODE}' already configured"
     exit 0
fi

CONFIG_PATH=${HOME}/${ES_UNZIP_FOLDER}/config

cp ${config_file} "${CONFIG_PATH}/elasticsearch.yml"

# Configure shards and replicas
update_property "index.number_of_shards" "${number_of_shards}"
update_property "index.number_of_replicas" "${number_of_replicas}"


# Take into account elk-all-in-one topo. Is consul agent present ?
if is_port_open "127.0.0.1" "8500"
then
    IS_CONSUL=0
    # Register the service in consul
    ES_BASE_DNS_SERVICE_NAME="${ES_SERVICE_NAME}.service.starlings"
    INSTANCE_ID="$(get_new_id service/${ES_SERVICE_NAME} retry |tail -1)"
    register_consul_service=${consul_utils}/register-consul-service.py
    chmod +x ${register_consul_service}
    log info "Register the service in consul: name=${ES_SERVICE_NAME}, port=9300, address=${ip_address}, tag=${INSTANCE_ID}"
    ${register_consul_service} -s ${ES_SERVICE_NAME} -p 9300 -a ${ip_address} -t ${INSTANCE_ID}
else
    IS_CONSUL=1
    log warning "Service Consul is not here."
fi

# Configure the cluster
update_property "cluster.name" "${cluster_name}"
if [[ "$IS_CONSUL" -eq "0" ]]
then
    update_property "node.name" "${INSTANCE_ID}.${ES_BASE_DNS_SERVICE_NAME}"
    update_property "network.publish_host" "${INSTANCE_ID}.${ES_BASE_DNS_SERVICE_NAME}"
    update_property "discovery.zen.ping.unicast.hosts" "${ES_BASE_DNS_SERVICE_NAME}"
else
    update_property "node.name" "${ES_SERVICE_NAME}_${ip_address}"
    update_property "network.publish_host" "${ip_address}"
fi

setServiceConfigured
log info "Elasticsearch end configuring"

#
# Curator crontab configuration
#

os_distribution="$(get_os_distribution)"
if [[ "${os_distribution}" == "ubuntu" ]]
then
    CURATOR_CMD="/usr/local/bin/curator"
    CRON_SERVICE_NAME="cron"
else
    CURATOR_CMD="/usr/bin/curator"
    CRON_SERVICE_NAME="crond"
fi

if [[ -s ${curator_cron_tab} ]]
then
    cp ${curator_cron_tab} /tmp/curator-crontab
else
    # Check parameters for close
    if [[ -n "$nb_close_older_than" && -n "$unit_close_older_than" ]]
    then
        log info "Set curator to close elasticsearch indices older than ${nb_close_older_than} ${unit_close_older_than}"
        cat >> /tmp/curator-crontab <<EOF
# Close elasticsearch indices older than ${nb_close_older_than} ${unit_close_older_than} at 02:00 every day
0 2 * * *    ${CURATOR_CMD} --master-only close indices --older-than ${nb_close_older_than} --time-unit ${unit_close_older_than} --timestring '\%Y.\%m.\%d'
EOF
    fi
    # Check parameters for delete
    if [[ -n "$nb_delete_older_than" && -n "$unit_delete_older_than" ]]
    then
        log info "Set curator delete elasticsearch indices older than ${nb_delete_older_than} ${unit_delete_older_than}"
        cat >> /tmp/curator-crontab <<EOF
# Delete elasticsearch indices older than ${nb_delete_older_than} ${unit_delete_older_than} at 03:00 every day
0 3 * * *    ${CURATOR_CMD} --master-only delete indices --older-than ${nb_delete_older_than} --time-unit ${unit_delete_older_than} --timestring '\%Y.\%m.\%d'
EOF
    fi
fi

if [[ -f "/tmp/curator-crontab" ]]
then
    sudo useradd curator
    sudo crontab -u curator /tmp/curator-crontab
    sudo service ${CRON_SERVICE_NAME} restart
    log info "Curator is configured"
else
    log info "Curator is not used"
fi

log end