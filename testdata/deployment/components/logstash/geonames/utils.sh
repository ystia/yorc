#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#

# Download archive containing geonames data
# params:
#   1 - file name without .zip suffix
#   2 - download repository
get_geonames_from_repository() {
    FNAME=$1
    REPOSITORY=$2

    TMP_TARGET="/tmp/geo"
    mkdir -p $TMP_TARGET
    cd ${TMP_TARGET}

    log info "Download ${FNAME}.zip from $REPOSITORY"
    if wget "${REPOSITORY}/${FNAME}.zip"
        then
            unzip ${FNAME}.zip

            # copy the data file to target
            #
            # this also allows to resolve the following issue :
            # be sure file is read because by default logstash ignors files
            # older than value defined by a configuration parameter named ignore_older ; this is by
            # default set to 24 hours
            log info "Copy ${FNAME}.txt to $GEONAMES_HOME/data"
            cp ${FNAME}.txt $GEONAMES_HOME/data
            rm -f ${TMP_TARGET}/*
            return 0
        else
            log error "Download error"
            rm -f ${TMP_TARGET}/*
            return 1
    fi
}
