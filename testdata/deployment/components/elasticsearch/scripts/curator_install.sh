#!/bin/bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

# Requirements:
# - This is ubuntu or centos based distribution
# - Cron is up and running
# - The host has access to the APT-YUM and the Pip repositories
#
# TODO: Throw errors when requirement are not met

. ${utils_scripts}/utils.sh

# Note: Same package name 'python-pip' on Ubuntu and CentOS
bash ${utils_scripts}/install-components.sh "python-pip" || {
    error_exit "Failed to install python-pip package !"
}

sudo -E pip install 'elasticsearch-curator<4'
