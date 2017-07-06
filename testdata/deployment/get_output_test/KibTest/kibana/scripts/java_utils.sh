#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

# Stores the JAVA_HOME in an enviroment file according to the JAVA component name
# params:
#   1- the JAVA_HOME value to store
#   2- the component name (usually the $SELF variable provided by alien)


function store_java_home () {
    JAVA_HOME=$1
    COMPONENT_NAME=$2
    if [[ ! -f ${HOME}/.starlings-java-installations.conf ]] ||
       [[ "$(grep -c ${COMPONENT_NAME} ${HOME}/.starlings-java-installations.conf)" == "0" ]]
    then
        echo "${COMPONENT_NAME}:${JAVA_HOME}" >> ${HOME}/.starlings-java-installations.conf
    fi
}

# Retrieves the JAVA_HOME value and export it as an environment property (it also modify the PATH variable to export $JAVA_HOME/bin).
# params:
#   1- the component name (usually the $HOST variable provided by alien)
function retrieve_java_home () {
    COMPONENT_NAME=$1
    if [[ -z "${COMPONENT_NAME}" ]]; then
        # Retrieve a default JVM if there is no component name (currently Alien doesn't support the HOST property)
        JAVA_HOME=$(head -1 ${HOME}/.starlings-java-installations.conf | cut -d: -f2)
    else
        JAVA_HOME=$(grep ${COMPONENT_NAME} ${HOME}/.starlings-java-installations.conf | cut -d: -f2)
    fi
    if [[ -z "${JAVA_HOME}" ]]
    then
        >&2 echo "Unable to retrieve JAVA_HOME for component ${COMPONENT_NAME}"
        exit 1
    else
        export JAVA_HOME="${JAVA_HOME}"
        export PATH="${JAVA_HOME}/bin:${PATH}"
    fi
}

# Check is the JAVA component is already installed
# params:
#   1- the component name
function is_java_already_installed() {
    COMPONENT_NAME=$1
    if [[ -f ${HOME}/.starlings-java-installations.conf ]]
    then
      if grep ${COMPONENT_NAME} ${HOME}/.starlings-java-installations.conf >/dev/null
      then
        return 0
      fi
    fi
    return 1
}
