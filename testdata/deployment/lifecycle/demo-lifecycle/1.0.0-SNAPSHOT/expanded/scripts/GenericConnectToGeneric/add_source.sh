#!/bin/bash
env_file=/tmp/$$.env
printenv > $env_file
wget --no-proxy --timeout=30 -S -q "http://a4c_registry/log_relation_operation.php?node=$TARGET_NODE&instance=$TARGET_INSTANCE&operation=add_source&tierNode=$SOURCE_NODE&tierInstance=$SOURCE_INSTANCE" --post-file=$env_file
exit 0
