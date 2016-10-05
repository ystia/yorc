#!/bin/bash

echo "install PHP5..."

NAME="PHP5"
LOCK="/tmp/lockaptget"

while true; do
  if mkdir "${LOCK}" &>/dev/null; then
    echo "$NAME take apt lock"
    break;
  fi
  echo "$NAME waiting apt lock to be released..."
  sleep 0.5
done

while sudo fuser /var/lib/dpkg/lock >/dev/null 2>&1 ; do
  echo "$NAME waiting for other software managers to finish..."
  sleep 0.5
done

sudo rm -f /var/lib/dpkg/lock
sudo apt-get update || (sleep 15; sudo apt-get update || exit ${1})
sudo apt-get -y -q install php5 php5-common php5-curl php5-cli php-pear php5-gd php5-mcrypt php5-xmlrpc php5-sqlite php-xml-parser || exit ${1}

rm -rf "${LOCK}"
echo "$NAME released apt lock"
