#!/bin/bash -e

echo "Using apt-get. Installing apache2 on one of the following : Debian, Ubuntu, Mint"
DEFAULT_PORT=80

NAME="Apache"
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
sudo apt-get install -y -q apache2 || exit ${1}

rm -rf "${LOCK}"
echo "$NAME released apt lock"

sudo /etc/init.d/apache2 stop
if [ ! -d $DOC_ROOT ]; then
  eval "sudo mkdir -p $DOC_ROOT"
fi
eval "sudo chown -R www-data:www-data $DOC_ROOT"

if [[ ("$PORT" == "$DEFAULT_PORT") ]]; then
  echo "Use default port for Apache : $DEFAULT_PORT"
else
  echo "Replacing port $DEFAULT_PORT with $PORT..."
  sudo sed -i -e "s/$DEFAULT_PORT/$PORT/g" /etc/apache2/ports.conf || exit ${1}
fi

echo "Change config of apache2"
if sudo test -f "/etc/apache2/sites-available/default"; then
  echo "Change the DocumentRoot of apache2 on Ubuntu < 14.04"
  sudo sed -i -e "s#DocumentRoot /var/www#DocumentRoot $DOC_ROOT#g" /etc/apache2/sites-available/default
fi
if sudo test -f "/etc/apache2/sites-available/000-default.conf"; then
  echo "Change the DocumentRoot of Apache2 on Ubuntu >= 14.04"
  sudo sed -i -e "s#DocumentRoot /var/www/html#DocumentRoot $DOC_ROOT#g" /etc/apache2/sites-available/000-default.conf
fi

sudo bash -c "echo ServerName localhost >> /etc/apache2/apache2.conf"

echo "Start apache2 whith new conf"
sudo /etc/init.d/apache2 start
echo "End of $0"
