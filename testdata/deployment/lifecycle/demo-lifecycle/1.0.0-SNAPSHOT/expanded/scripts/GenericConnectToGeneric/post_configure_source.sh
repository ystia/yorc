#!/bin/bash
env_file=/tmp/$$.env
printenv > $env_file
wget --no-proxy --timeout=30 -S -q "http://a4c_registry/log_relation_operation.php?node=$SOURCE_NODE&instance=$SOURCE_INSTANCE&operation=post_configure_source&tierNode=$TARGET_NODE&tierInstance=$TARGET_INSTANCE" --post-file=$env_file
exit 0
