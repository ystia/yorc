#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

echo "------------------------ ENV ------------------------"
echo "ENV VAR USED scripts : ${scripts}"
echo "ENV VAR USED PORT    : ${PORT}"
echo "-----------------------------------------------------"

SAMPLEWEBSERVER_SRC=SampleWebServer.py
SAMPLEWEBSERVER_LOGS=${scripts}/SampleWebServer.logs
SAMPLEWEBSERVER_PID=${scripts}/SampleWebServer.pid

source ${utils_scripts}/utils.sh

echo "Start python ${scripts}/${SAMPLEWEBSERVER_SRC} on port ${PORT}"
nohup python ${scripts}/${SAMPLEWEBSERVER_SRC} ${PORT} >> ${SAMPLEWEBSERVER_LOGS} 2>&1 &
echo "$!" > ${SAMPLEWEBSERVER_PID}

wait_for_port_to_be_open "127.0.0.1" "${PORT}"
exit $?
