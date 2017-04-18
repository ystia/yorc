#!/usr/bin/env bash

#
# Starlings
# Copyright (C) 2016 Bull S.A.S. - All rights reserved
#

# Installation of packages on CentOS

echo "Using yum. Installing $@ on CentOS"

sudo yum -y install $@ || exit 2

echo "Successfully installed $@ on CentOS"

