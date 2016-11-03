#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

# Enviroment variables
# LOGSTASH_HOME: Logstash install home directory
# KAFKA_ZK_BASE_DNS_SRV_NAME: ZooKeeper Service base DNS name

source ${utils_scripts}/utils.sh
ensure_home_var_is_set

lock "$(basename $0)"

flag="${STARLINGS_DIR}/.${SOURCE_NODE}-preconfiguresource-input-Flag"

if [[ -e "${flag}" ]]; then
    log info "Kafka input already configured, skipping."
    unlock "$(basename $0)"
    exit 0
fi


source $HOME/.starlings/${SOURCE_NODE}-service.env

mkdir -p ${LOGSTASH_HOME}/conf

source ${ls_scripts}/java_utils.sh

retrieve_java_home "${HOST}"
LS_JAVA_OPTS=${JAVA_OPTS}
${LOGSTASH_HOME}/bin/plugin install logstash-input-kafka
cat > ${LOGSTASH_HOME}/conf/1-kafka_logstash_inputs.conf << END
input {
    kafka {
        topic_id => "${TOPIC_NAME}"
        zk_connect => "kafka-zk.service.starlings:2181"
        group_id => "${SOURCE_NODE}"
    }
}
END

touch "${flag}"
unlock "$(basename $0)"
