{
    "working_directory": "work",
    "workers_number": 3,
    "os_auth_url": "${ks_url}",
    "os_tenant_name": "${ks_tenant}",
    "os_user_name": "${ks_user}",
    "os_password": "${ks_password}",
    "os_region": "${region}",
    "os_private_network_name": "${priv_net}",
    "os_prefix": "${prefix}",
    "os_default_security_groups": ["default", "${secgrp}"],
    "telemetry": {
        "statsd_address": "${statsd_ip}:8125",
        "expose_prometheus_endpoint": true
    }
}
