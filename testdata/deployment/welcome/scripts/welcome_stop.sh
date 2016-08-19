#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

PID=`ps -aux |grep 'SampleWebServer.py' |grep -v 'grep' |awk '{ print $2 }'`
if [ -n "$PID" ]
then
    kill -15 $PID
fi
