.. _janus_config_section:

Janus Server Configuration
==========================

Janus has various configuration options that could be specified either by command-line flags, configuration file or environment variables.

If an option is specified several times using flags, environment and config file, command-line flag will have the precedence then the environment variable and finally the value defined in the configuration file. 

Command-line options
--------------------
.. _option_config_cmd:

  * ``--config`` or ``-c``: Specify an alternative configuration file. By default Janus will look for a file named config.janus.json in ``/etc/janus`` directory then if not found in the current directory.

.. _option_consul_addr_cmd:

  * ``--consul_address``: Specify the address (using the format host:port) of Consul. Consul default is used if not provided.

.. _option_consul_token_cmd:

  * ``--consul_token``: Specify the security token to use with Consul. No security token used by default.

.. _option_consul_dc_cmd:

  * ``--consul_datacenter``: Specify the Consul's datacenter to use. Consul default (dc1) is used by default.

.. _option_pub_routines_cmd:

  * ``--consul_publisher_max_routines``: Maximum number of parallelism used to store key/values in Consul. If you increase the default value you may need to tweak the ulimit max open files. If set to 0 or less the default value (500) will be used.

.. _option_shut_timeout_cmd:

  * ``--graceful_shutdown_timeout``: Timeout to wait for a graceful shutdown of the Janus server. After this delay the server immediately exits. The default is ``5m``.

.. _option_http_addr_cmd:

  * ``--http_address``: Restrict the listening interface for the Janus HTTP REST API. By default Janus listens on all available interfaces

.. _option_http_port_cmd:

  * ``--http_port``: Port number for the Janus HTTP REST API. If omitted or set to '0' then the default port number is used, any positive integer will be used as it, and finally any negative value will let use a random port.

.. _option_keyfile_cmd:

  * ``--key_file``: File path to a PEM-encoded private key. The key is used to enable SSL for the Janus HTTP REST API. This must be provided along with cert_file. If one of key_file or cert_file is not provided then SSL is disabled.

.. _option_certfile_cmd:

  * ``--cert_file``: File path to a PEM-encoded certificate. The certificate is used to enable SSL for the Janus HTTP REST API. This must be provided along with key_file. If one of key_file or cert_file is not provided then SSL is disabled.

.. _option_os_authurl_cmd:

  * ``--os_auth_url``: Specify the authentication url for OpenStack (should be the Keystone endpoint ie: http://your-openstack:5000/v2.0). There is no default for this option.

.. _option_os_tenantid_cmd:

  * ``--os_tenant_id``: Specify the OpenStack tenant id to use. Either this or ``--os_tenant_name`` should be provided. There is no default for this option.

.. _option_os_tenantname_cmd:

  * ``--os_tenant_name``: Specify the OpenStack tenant name to use. Either this or ``--os_tenant_id`` should be provided. There is no default for this option.

.. _option_os_username_cmd:

  * ``--os_user_name``: Specify the OpenStack user name to use. There is no default for this option.

.. _option_os_password_cmd:

  * ``--os_password``: Specify the OpenStack password to use. There is no default for this option.

.. _option_os_region_cmd:

  * ``--os_region``: Specify the OpenStack region to use. Defaults to ``RegionOne``.

.. _option_os_prefix_cmd:

  * ``--os_prefix``: Specify a prefix that will be used for names when creating resources such as Compute instances or volumes. Defaults to ``janus-``.

.. _option_os_privatenet_cmd:

  * ``--os_private_network_name``: Specify the name of private network to use as primary adminstration network between Janus and Compute instances. It should be a private network accessible by this instance of Janus.

.. _option_os_secgroups_cmd:

  * ``--os_default_security_groups``: Default security groups to be used when creating a Compute instance. It could be a comma-separated list of security group names or this option may be specified several times.

.. _option_pluginsdir_cmd:

  * ``--plugins_directory``: The name of the plugins directory of the Janus server. The default is to use a directory named *plugins* in the current directory.

.. _option_workers_cmd:

  * ``--workers_number``: Janus instances use a pool of workers to handle deployment tasks. This option defines the size of this pool. If not set the default value of `3` will be used.

.. _option_workdir_cmd: 

  * ``--working_directory`` or ``-w``: Specify an alternative working directory for Janus. The default is to use a directory named *work* in the current directory.

Configuration files
-------------------

Configuration files are JSON-formatted as a single JSON object containing the following configuration options. 
By default Janus will look for a file named config.janus.json in ``/etc/janus`` directory then if not found in the current directory. 
The :ref:`--config <option_config_cmd>` command line flag allows to specify an alternative configuration file.

Bellow is an example of configuration file.

.. code-block:: JSON
    
    {
        "os_auth_url": "http://your-openstack:5000/v2.0",
        "os_tenant_name": "your-tenant",
        "os_user_name": "os-user",
        "os_password": "os-password",
        "os_prefix": "janus1-",
        "os_private_network_name": "default-private-network",
        "os_default_security_groups": ["default"]
    }


Bellow is an example of configuration file with TLS enable.

.. code-block:: JSON
    
    {
        "os_auth_url": "http://your-openstack:5000/v2.0",
        "os_tenant_name": "your-tenant",
        "os_user_name": "os-user",
        "os_password": "os-password",
        "os_prefix": "janus1-",
        "os_private_network_name": "default-private-network",
        "os_default_security_groups": ["default"],
        "key_file": "/etc/pki/tls/private/janus.key",
        "cert_file": "/etc/pki/tls/certs/janus.crt"
    }


.. _option_consul_addr_cfg:

  * ``consul_address``: Equivalent to :ref:`--consul_address <option_consul_addr_cmd>` command-line flag.

.. _option_consul_token_cfg:

  * ``consul_token``: Equivalent to :ref:`--consul_token <option_consul_token_cmd>` command-line flag.

.. _option_consul_dc_cfg:

  * ``consul_datacenter``: Equivalent to :ref:`--consul_datacenter <option_consul_dc_cmd>` command-line flag.

.. _option_pub_routines_cfg:

  * ``consul_publisher_max_routines``: Equivalent to :ref:`--consul_publisher_max_routines <option_pub_routines_cmd>` command-line flag.

.. _option_shut_timeout_cfg:

  * ``server_graceful_shutdown_timeout``: Equivalent to :ref:`--graceful_shutdown_timeout <option_shut_timeout_cmd>` command-line flag.

.. _option_http_addr_cfg:

  * ``http_address``: Equivalent to :ref:`--http_address <option_http_addr_cmd>` command-line flag.

.. _option_http_port_cfg:

  * ``http_port``: Equivalent to :ref:`--http_port <option_http_port_cmd>` command-line flag.

.. _option_keyfile_cfg:

  * ``key_file``: Equivalent to :ref:`--key_file <option_keyfile_cmd>` command-line flag.

.. _option_certfile_cfg:

  * ``cert_file``: Equivalent to :ref:`--cert_file <option_certfile_cmd>` command-line flag.

.. _option_os_authurl_cfg:

  * ``os_auth_url``: Equivalent to :ref:`--os_auth_url <option_os_authurl_cmd>` command-line flag.

.. _option_os_tenantid_cfg:

  * ``os_tenant_id``: Equivalent to :ref:`--os_tenant_id <option_os_tenantid_cmd>` command-line flag.

.. _option_os_tenantname_cfg:

  * ``os_tenant_name``: Equivalent to :ref:`--os_tenant_name <option_os_tenantname_cmd>` command-line flag.

.. _option_os_username_cfg:

  * ``os_user_name``: Equivalent to :ref:`--os_user_name <option_os_username_cmd>` command-line flag.

.. _option_os_password_cfg:

  * ``os_password``: Equivalent to :ref:`--os_password <option_os_password_cmd>` command-line flag.

.. _option_os_region_cfg:

  * ``os_region``: Equivalent to :ref:`--os_region <option_os_region_cmd>` command-line flag.

.. _option_os_prefix_cfg:

  * ``os_prefix``: Equivalent to :ref:`--os_prefix <option_os_prefix_cmd>` command-line flag.

.. _option_os_privatenet_cfg:

  * ``os_private_network_name``: Equivalent to :ref:`--os_private_network_name <option_os_privatenet_cmd>` command-line flag.

.. _option_os_secgroups_cfg:

  * ``os_default_security_groups``: Equivalent to :ref:`--os_default_security_groups <option_os_secgroups_cmd>` command-line flag.

.. _option_plugindir_cfg:

  * ``plugins_directory``: Equivalent to :ref:`--plugins_directory <option_pluginsdir_cmd>` command-line flag.

.. _option_workers_cfg:

  * ``workers_number``: Equivalent to :ref:`--workers_number <option_workers_cmd>` command-line flag.

.. _option_workdir_cfg: 

  * ``working_directory``: Equivalent to :ref:`--working_directory <option_workdir_cmd>` command-line flag.
 

Environment variables
---------------------

.. _option_consul_addr_env:

  * ``JANUS_CONSUL_ADDRESS``: Equivalent to :ref:`--consul_address <option_consul_addr_cmd>` command-line flag.

.. _option_consul_token_env:

  * ``JANUS_CONSUL_TOKEN``: Equivalent to :ref:`--consul_token <option_consul_token_cmd>` command-line flag.

.. _option_consul_dc_env:

  * ``JANUS_CONSUL_DATACENTER``: Equivalent to :ref:`--consul_datacenter <option_consul_dc_cmd>` command-line flag.

.. _option_pub_routines_env:

  * ``JANUS_CONSUL_PUBLISHER_MAX_ROUTINES``: Equivalent to :ref:`--consul_publisher_max_routines <option_pub_routines_cmd>` command-line flag.

.. _option_shut_timeout_env:

  * ``JANUS_SERVER_GRACEFUL_SHUTDOWN_TIMEOUT``: Equivalent to :ref:`--graceful_shutdown_timeout <option_shut_timeout_cmd>` command-line flag.

.. _option_http_addr_env:

  * ``JANUS_HTTP_ADDRESS``: Equivalent to :ref:`--http_address <option_http_addr_cmd>` command-line flag.

.. _option_http_port_env:

  * ``JANUS_HTTP_PORT``: Equivalent to :ref:`--http_port <option_http_port_cmd>` command-line flag.

.. _option_keyfile_env:

  * ``JANUS_KEY_FILE``: Equivalent to :ref:`--key_file <option_keyfile_cmd>` command-line flag.

.. _option_certfile_env:

  * ``JANUS_CERT_FILE``: Equivalent to :ref:`--cert_file <option_certfile_cmd>` command-line flag.

.. _option_os_authurl_env:

  * ``OS_AUTH_URL``: Equivalent to :ref:`--os_auth_url <option_os_authurl_cmd>` command-line flag.

.. _option_os_tenantid_env:

  * ``OS_TENANT_ID``: Equivalent to :ref:`--os_tenant_id <option_os_tenantid_cmd>` command-line flag.

.. _option_os_tenantname_env:

  * ``OS_TENANT_NAME``: Equivalent to :ref:`--os_tenant_name <option_os_tenantname_cmd>` command-line flag.

.. _option_os_username_env:

  * ``OS_USER_NAME``: Equivalent to :ref:`--os_user_name <option_os_username_cmd>` command-line flag.

.. _option_os_password_env:

  * ``OS_PASSWORD``: Equivalent to :ref:`--os_password <option_os_password_cmd>` command-line flag.

.. _option_os_region_env:

  * ``OS_REGION``: Equivalent to :ref:`--os_region <option_os_region_cmd>` command-line flag.

.. _option_os_prefix_env:

  * ``JANUS_OS_PREFIX``: Equivalent to :ref:`--os_prefix <option_os_prefix_cmd>` command-line flag.

.. _option_os_privatenet_env:

  * ``JANUS_OS_PRIVATE_NETWORK_NAME``: Equivalent to :ref:`--os_private_network_name <option_os_privatenet_cmd>` command-line flag.

.. _option_os_secgroups_env:

  * ``JANUS_OS_DEFAULT_SECURITY_GROUPS``: Equivalent to :ref:`--os_default_security_groups <option_os_secgroups_cmd>` command-line flag.

.. _option_plugindir_env:

  * ``JANUS_PLUGIN_DIRECTORY``: Equivalent to :ref:`--plugins_directory <option_pluginsdir_cmd>` command-line flag.

.. _option_workers_env:

  * ``JANUS_WORKERS_NUMBER``: Equivalent to :ref:`--workers_number <option_workers_cmd>` command-line flag.

.. _option_workdir_env: 

  * ``JANUS_WORKING_DIRECTORY``: Equivalent to :ref:`--working_directory <option_workdir_cmd>` command-line flag.

.. _option_log_env: 

  * ``JANUS_LOG``: If set to ``1`` or ``DEBUG``, enables debug logging for Janus.
 