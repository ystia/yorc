#!/usr/bin/env bash
#
# Starlings
# Copyright (C) 2015 Bull S.A.S. - All rights reserved
#

# Installation of packages on Ubuntu


scriptDir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

LOCK="/tmp/lockaptget"


function purge_lock_and_exit () {
    rm -rf "${LOCK}"
    exit ${1}
}

function shouldWeUpdateApt() {
  if [[ -e /tmp/forceaptgetupdate ]]; then
    sudo rm -f /tmp/forceaptgetupdate
    echo "true"
    return
  fi

  local lastUpdate=$(stat -c %Y '/var/lib/apt/periodic/update-success-stamp')
  local currentDate=$(date +'%s')

  delay=$((currentDate - lastUpdate))
  # Do not perform apt-get update if last update is less than an hour
  maxInterval=$((60 * 60))

  if [[ ${delay} -le ${maxInterval} ]]
  then
    echo "false"
  else
    echo "true"
  fi
}

echo "Using apt-get. Installing $@ on Ubuntu"

while true; do
  if mkdir "${LOCK}" &>/dev/null; then
    echo "Got the lock"
    break;
  fi
  echo "Waiting the end of one of our recipes..."
  sleep 0.5
done

while sudo fuser /var/lib/dpkg/lock >/dev/null 2>&1 ; do
  echo "Waiting for other software managers to finish..."
  sleep 0.5
done

sudo rm -f /var/lib/dpkg/lock
if [[ "$(shouldWeUpdateApt)" == "true" ]]
then
  sudo -E apt-get update || (sleep 15; sudo -E apt-get update || purge_lock_and_exit 1)
fi
sudo -E DEBIAN_FRONTEND=noninteractive apt-get install -y -q $@ || (sleep 15; sudo -E DEBIAN_FRONTEND=noninteractive apt-get install -y -q $@  || purge_lock_and_exit 2)
rm -rf "${LOCK}"

echo "Successfully installed $@ on Ubuntu"
