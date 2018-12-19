{
    "working_directory": "work",
    "workers_number": 30,
    "server_id": "${server_id}",
    "resources_prefix": "${prefix}",
    "infrastructures":{
          "openstack": {
            "auth_url": "${ks_url}",
            "tenant_name": "${ks_tenant}",
            "user_name": "${ks_user}",
            "password": "${ks_password}",
            "region": "${region}",
            "private_network_name": "${priv_net}",
            "default_security_groups": ["default", "${secgrp}"]
          }
    },
    "telemetry": {
        "statsd_address": "${statsd_ip}:8125",
        "expose_prometheus_endpoint": true
    }
}
