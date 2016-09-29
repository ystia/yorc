#!/bin/bash

# loop until the php scripts are available
result=$(wget --no-proxy --timeout=30 -qO- http://a4c_registry/echo.php)
while [ "$result" != "OK" ]; do
  sleep 5
  result=$(wget --no-proxy -qO- http://a4c_registry/echo.php)
done

env_file=/tmp/$$.env
printenv > $env_file
wget --no-proxy -S -q "http://a4c_registry/log_node_operation.php?node=$NODE&instance=$INSTANCE&operation=create" --post-file=$env_file
exit 0
