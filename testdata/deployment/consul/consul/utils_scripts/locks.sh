#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#

readonly LOCKFILE_DIR=/var/lock/starlings
readonly LOCK_FD=200

lock() {
    local prefix=$1
    local fd=${2:-$LOCK_FD}
    local lock_file=$LOCKFILE_DIR/$prefix.lock
    # Need sudo + chown on centos (not on ubuntu)
    sudo mkdir -p ${LOCKFILE_DIR}
    sudo chown $(id -un):$(id -gn) ${LOCKFILE_DIR}
    # create lock file
    eval "exec ${fd}>${lock_file}"

    # acquire the lock
    flock -x ${fd}
}


unlock() {
    local prefix=$1
    local fd=${2:-$LOCK_FD}
    local lock_file=$LOCKFILE_DIR/$prefix.lock
    mkdir -p ${LOCKFILE_DIR}

    # acquire the lock
    flock -u ${fd}
}