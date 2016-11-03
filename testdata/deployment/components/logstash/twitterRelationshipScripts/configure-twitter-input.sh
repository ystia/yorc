#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

# Enviroment variables
# LOGSTASH_HOME: Logstash install home directory

source ${utils_scripts}/utils.sh
ensure_home_var_is_set
log begin

lock "$(basename $0)"

flag="${STARLINGS_DIR}/.${TARGET_NODE}-preconfiguresource-twitter-input-Flag"

if [[ -e "${flag}" ]]; then
    log info "Twitter input of ${TARGET_NODE} already configured, skipping."
    unlock "$(basename $0)"
    exit 0
fi


source $HOME/.starlings/${TARGET_NODE}-service.env

mkdir -p ${LOGSTASH_HOME}/conf

# Check the TwitterConnector properties validity
if [[ "${USE_SAMPLES}" = "false" ]] && [[ -z "${KEYWORDS}" ]] && [[ -z "${FOLLOWS}" ]]
then
    error_exit "ERROR: One of the TwitterConnector properties keywords, follows must be set when use_samples is not set"
fi

if [[ "${USE_PROXY}" = "true" ]]
then
    if [[ -n "${http_proxy}" ]]
    then
        DEFAULT_PROXY_ADDRESS=$(echo ${http_proxy#*://} | cut -f1 -d:)
        DEFAULT_PROXY_PORT=$(echo ${http_proxy#*://} | cut -f2 -d:)
    fi
    if [[ -n "${https_proxy}" ]]
    then
        DEFAULT_PROXY_ADDRESS=$(echo ${https_proxy#*://} | cut -f1 -d:)
        DEFAULT_PROXY_PORT=$(echo ${https_proxy#*://} | cut -f2 -d:)
    fi
    if [[ -z "${PROXY_ADDRESS}" ]]
    then
       PROXY_ADDRESS=${DEFAULT_PROXY_ADDRESS}
    fi
    if [[ -z "${PROXY_PORT}" ]]
    then
       PROXY_PORT=${DEFAULT_PROXY_PORT}
    fi
    if [[ -z "${PROXY_ADDRESS}" ]] || [[ -z "${PROXY_PORT}" ]]
    then
        error_exit "ERROR: No values for proxy_address and/or proxy_port"
    fi
fi

cat > ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
input {
    twitter {
        consumer_key => "${CONSUMER_KEY}"
        consumer_secret => "${CONSUMER_SECRET}"
        oauth_token => "${OAUTH_TOKEN}"
        oauth_token_secret => "${OAUTH_TOKEN_SECRET}"
        use_proxy => ${USE_PROXY}
END

if [[ "${USE_PROXY}" = "true" ]]
then
cat >> ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
        proxy_address => "${PROXY_ADDRESS}"
        proxy_port => ${PROXY_PORT}
END
fi

if [[ -n "${KEYWORDS}" ]]
then
cat >> ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
        keywords => ${KEYWORDS}
END
fi
if [[ -n "${FOLLOWS}" ]]
then
cat >> ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
        follows => ${FOLLOWS}
END
fi
if [[ -n "${LOCATIONS}" ]]
then
cat >> ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
        locations => "${LOCATIONS}"
END
fi
if [[ -n "${LANGUAGES}" ]]
then
cat >> ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
        languages => ${LANGUAGES}
END
fi

if [[ -n "${TAGS}" ]]
then
cat >> ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
        tags => ${TAGS}
END
fi

cat >> ${LOGSTASH_HOME}/conf/1-${SOURCE_NODE}_logstash_inputs.conf << END
        use_samples => ${USE_SAMPLES}
        full_tweet => ${FULL_TWEET}
        ignore_retweets => ${IGNORE_RETWEETS}
    }
}
END


touch "${flag}"
unlock "$(basename $0)"

log end "Logstash Twitter input of ${TARGET_NODE} successfully configured"
