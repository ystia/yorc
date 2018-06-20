.. _yorc_config_section:

Yorc Server Configuration
==========================

Yorc has various configuration options that could be specified either by command-line flags, configuration file or environment variables.

If an option is specified several times using flags, environment and config file, command-line flag will have the precedence then the environment variable and finally the value defined in the configuration file. 

Globals Command-line options
----------------------------

.. _option_ansible_ssh_cmd:

  * ``--ansible_use_openssh``: Prefer OpenSSH over Paramiko a Python implementation of SSH (the default) to provision remote hosts. OpenSSH have several optimization like reusing connections that should improve preformance but may lead to issues on older systems. 

.. _option_ansible_debug_cmd:

  * ``--ansible_debug``: Prints massive debug information from Ansible especially about connections

.. _option_ansible_connection_retries_cmd:

  * ``--ansible_connection_retries``: Number of retries in case of Ansible SSH connection failure.

.. _option_operation_remote_base_dir_cmd:

  * ``--operation_remote_base_dir``: Specify an alternative working directory for Ansible on provisioned Compute.

.. _option_config_cmd:

  * ``--config`` or ``-c``: Specify an alternative configuration file. By default Yorc will look for a file named config.yorc.json in ``/etc/yorc`` directory then if not found in the current directory.

.. _option_consul_addr_cmd:

  * ``--consul_address``: Specify the address (using the format host:port) of Consul. Consul default is used if not provided.

.. _option_consul_token_cmd:

  * ``--consul_token``: Specify the security token to use with Consul. No security token used by default.

.. _option_consul_dc_cmd:

  * ``--consul_datacenter``: Specify the Consul's datacenter to use. Consul default (dc1) is used by default.

.. _option_consul_key_cmd:

  * ``--consul_key_file``: Specify the Consul client's key to use when commuicating over TLS.

.. _option_consul_cert_cmd:

  * ``--consul_cert_file``: Specify the Consul client's certificate to use when commuicating over TLS.

.. _option_consul_ca_cert_cmd:

  * ``--consul_ca_cert``: Specify the CA used to sign Consul certificates.

.. _option_consul_ca_path_cmd:

  * ``--consul_ca_path``: Specify the path to the CA used to sign Consul certificates

.. _option_consul_ssl_cmd:

  * ``--consul_ssl``: If set to true, enable SSL (false by default).

.. _option_consul_ssl_verify_cmd:

  * ``--consul_ssl_verify``: If set to false, disable Consul certificate checking (true by default is ssl enabled).

.. _option_pub_routines_cmd:

  * ``--consul_publisher_max_routines``: Maximum number of parallelism used to store key/values in Consul. If you increase the default value you may need to tweak the ulimit max open files. If set to 0 or less the default value (500) will be used.

.. _option_shut_timeout_cmd:

  * ``--graceful_shutdown_timeout``: Timeout to wait for a graceful shutdown of the Yorc server. After this delay the server immediately exits. The default is ``5m``.

.. _option_wf_step_termination_timeout_cmd:

  * ``--wf_step_graceful_termination_timeout``: Timeout to wait for a graceful termination of a workflow step during concurrent workflow step failure. After this delay the step is set on error. The default is ``2m``.

.. _option_http_addr_cmd:

  * ``--http_address``: Restrict the listening interface for the Yorc HTTP REST API. By default Yorc listens on all available interfaces

.. _option_http_port_cmd:

  * ``--http_port``: Port number for the Yorc HTTP REST API. If omitted or set to '0' then the default port number is used, any positive integer will be used as it, and finally any negative value will let use a random port.

.. _option_keep_remote_path_cmd:

  * ``--keep_operation_remote_path``: If set to true, do not delete temporary artifacts on provisioned Compute at the end of deployment (false by default for deployment temporary artifacts cleanup).

.. _option_keyfile_cmd:

  * ``--key_file``: File path to a PEM-encoded private key. The key is used to enable SSL for the Yorc HTTP REST API. This must be provided along with cert_file. If one of key_file or cert_file is not provided then SSL is disabled.

.. _option_certfile_cmd:

  * ``--cert_file``: File path to a PEM-encoded certificate. The certificate is used to enable SSL for the Yorc HTTP REST API. This must be provided along with key_file. If one of key_file or cert_file is not provided then SSL is disabled.

.. _option_ca_file_cmd:

  * ``--ca_file``: If set to true, enable TLS certificate checking. Must be provided with cert_file ; key_file and ca_file. Disabled by default.

.. _option_ssl_verify_cmd:

  * ``--ssl_verify``: If set to true, enable TLS certificate checking for clients of the Yorc's API. Must be provided with cert_file ; key_file and ca_file. Disabled by default.

.. _option_pluginsdir_cmd:

  * ``--plugins_directory``: The name of the plugins directory of the Yorc server. The default is to use a directory named *plugins* in the current directory.

.. _option_resources_prefix_cmd:

  * ``--resources_prefix``: Specify a prefix that will be used for names when creating resources such as Compute instances or volumes. Defaults to ``yorc-``.

.. _option_workers_cmd:

  * ``--workers_number``: Yorc instances use a pool of workers to handle deployment tasks. This option defines the size of this pool. If not set the default value of `3` will be used.

.. _option_workdir_cmd: 

  * ``--working_directory`` or ``-w``: Specify an alternative working directory for Yorc. The default is to use a directory named *work* in the current directory.

.. _option_server_id_cmd:

  * ``--server_id``: Specify the server ID used to identify the server node in a cluster. The default is ``server_0``.


.. _yorc_config_file_section:

Configuration files
-------------------

Configuration files are either JSON or YAML formatted as a single object containing the following configuration options. 
By default Yorc will look for a file named config.yorc.json in ``/etc/yorc`` directory then if not found in the current directory. 
The :ref:`--config <option_config_cmd>` command line flag allows to specify an alternative configuration file.

Below is an example of configuration file.

.. code-block:: JSON
    
    {
      "resources_prefix": "yorc1-",
      "infrastructures": {
        "openstack": {
          "auth_url": "http://your-openstack:5000/v2.0",
          "tenant_name": "your-tenant",
          "user_name": "os-user",
          "password": "os-password",
          "private_network_name": "default-private-network",
          "default_security_groups": ["default"]
        }
      }
    }


Below is an example of configuration file with TLS enabled.

.. code-block:: JSON
    
    {
      "resources_prefix": "yorc1-",
      "key_file": "/etc/pki/tls/private/yorc.key",
      "cert_file": "/etc/pki/tls/certs/yorc.crt",
      "infrastructures": {
        "openstack": {
          "auth_url": "http://your-openstack:5000/v2.0",
          "tenant_name": "your-tenant",
          "user_name": "os-user",
          "password": "os-password",
          "private_network_name": "default-private-network",
          "default_security_groups": ["default"]
        }
      }
    }

.. _option_shut_timeout_cfg:

  * ``server_graceful_shutdown_timeout``: Equivalent to :ref:`--graceful_shutdown_timeout <option_shut_timeout_cmd>` command-line flag.

.. _option_wf_step_termination_timeout_cfg:

  * ``wf_step_graceful_termination_timeout``: Equivalent to :ref:`--wf_step_graceful_termination_timeout <option_wf_step_termination_timeout_cmd>` command-line flag.

.. _option_http_addr_cfg:

  * ``http_address``: Equivalent to :ref:`--http_address <option_http_addr_cmd>` command-line flag.

.. _option_http_port_cfg:

  * ``http_port``: Equivalent to :ref:`--http_port <option_http_port_cmd>` command-line flag.

.. _option_keyfile_cfg:

  * ``key_file``: Equivalent to :ref:`--key_file <option_keyfile_cmd>` command-line flag.

.. _option_certfile_cfg:

  * ``cert_file``: Equivalent to :ref:`--cert_file <option_certfile_cmd>` command-line flag.

.. _option_sslverify_cfg:

  * ``ssl_verify``: Equivalent to :ref:`--ssl_verify <option_sslverify_cmd>` command-line flag.

.. _option_cafile_cfg:

  * ``ca_file``: Equivalent to :ref:`--ca_file <option_cafile_cmd>` command-line flag.

.. _option_plugindir_cfg:

  * ``plugins_directory``: Equivalent to :ref:`--plugins_directory <option_pluginsdir_cmd>` command-line flag.

.. _option_resources_prefix_cfg:

  * ``resources_prefix``: Equivalent to :ref:`--resources_prefix <option_resources_prefix_cmd>` command-line flag.

.. _option_workers_cfg:

  * ``workers_number``: Equivalent to :ref:`--workers_number <option_workers_cmd>` command-line flag.

.. _option_workdir_cfg: 

  * ``working_directory``: Equivalent to :ref:`--working_directory <option_workdir_cmd>` command-line flag.

.. _option_server_id_cfg:

  * ``server_id``: Equivalent to :ref:`--server_id <option_server_id_cmd>` command-line flag.

.. _yorc_config_file_ansible_section:

Ansible configuration
~~~~~~~~~~~~~~~~~~~~~

Below is an example of configuration file with Ansible configuration options.

.. code-block:: JSON
    
    {
      "resources_prefix": "yorc1-",
      "infrastructures": {
        "openstack": {
          "auth_url": "http://your-openstack:5000/v2.0",
          "tenant_name": "your-tenant",
          "user_name": "os-user",
          "password": "os-password",
          "private_network_name": "default-private-network",
          "default_security_groups": ["default"]
        }
      },
      "ansible": {
        "use_openssh": true,
        "connection_retries": 3,
        "hosted_operations": {
          "unsandboxed_operations_allowed": false,                                     
          "default_sandbox": {                               
            "image": "jfloff/alpine-python:2.7-slim",  
            "entrypoint": ["python", "-c"],
            "command": ["import time;time.sleep(31536000);"]                                                   
          }            
        }  
      }
    }

All available configuration options for Ansible are:

.. _option_ansible_ssh_cfg:

  * ``use_openssh``: Equivalent to :ref:`--ansible_use_openssh <option_ansible_ssh_cmd>` command-line flag.

.. _option_ansible_debug_cfg:

  * ``debug``: Equivalent to :ref:`--ansible_debug <option_ansible_debug_cmd>` command-line flag.

.. _option_ansible_connection_retries_cfg:

  * ``connection_retries``: Equivalent to :ref:`--ansible_connection_retries <option_ansible_connection_retries_cmd>` command-line flag.

.. _option_operation_remote_base_dir_cfg:

  * ``operation_remote_base_dir``: Equivalent to :ref:`--operation_remote_base_dir <option_operation_remote_base_dir_cmd>` command-line flag.

.. _option_keep_remote_path_cfg:

  * ``keep_operation_remote_path``: Equivalent to :ref:`--keep_operation_remote_path <option_keep_remote_path_cmd>` command-line flag.

.. _option_ansible_sandbox_hosted_ops_cfg:

  * ``hosted_operations``: This is a complex structure that allow to define the behavior of a Yorc server when it executes an hosted operation.
    For more information about hosted operation please see :ref:`The hosted operations paragraph in the TOSCA support section <tosca_orchestrator_hosted_operations>`.
    This structure contains the following configuration options:

    .. _option_ansible_sandbox_hosted_ops_unsandboxed_flag_cfg:

    * ``unsandboxed_operations_allowed``: This option control if operations can be executed directly on the system that hosts Yorc if no default sandbox is defined. **This is not permitted by default.** 

    .. _option_ansible_sandbox_hosted_ops_default_sandbox_cfg:

    * ``default_sandbox``: This complex structure allows to define the default docker container to use to sandbox orchestrator-hosted operations.
      Bellow configuration options ``entrypoint`` and ``command`` should be carefully set to run the container and make it sleep until operations are executed on it.
      Defaults options will run a python inline script that sleeps for 1 year.

      .. _option_ansible_sandbox_hosted_ops_default_sandbox_image_cfg:

      * ``image``: This is the docker image identifier (in the docker format ``[repository/]name[:tag]``) is option is **required**.

      .. _option_ansible_sandbox_hosted_ops_default_sandbox_entrypoint_cfg:

      * ``entrypoint``: This allows to override the default image entrypoint. If both ``entrypoint`` and ``command`` are empty the default value for ``entrypoint`` is ``["python", "-c"]``.

      .. _option_ansible_sandbox_hosted_ops_default_sandbox_command_cfg:

      * ``command``: This allows to run a command within the container.  If both ``entrypoint`` and ``command`` are empty the default value for ``command`` is ``["import time;time.sleep(31536000);"]``.

      .. _option_ansible_sandbox_hosted_ops_default_sandbox_env_cfg:

      * ``env``: An optional list environment variables to set when creating the container. The format of each variable is ``var_name=value``.

.. _yorc_config_file_consul_section:

Consul configuration
~~~~~~~~~~~~~~~~~~~~

Below is an example of configuration file with Consul configuration options.

.. code-block:: JSON
    
    {
      "resources_prefix": "yorc1-",
      "infrastructures": {
        "openstack": {
          "auth_url": "http://your-openstack:5000/v2.0",
          "tenant_name": "your-tenant",
          "user_name": "os-user",
          "password": "os-password",
          "private_network_name": "default-private-network",
          "default_security_groups": ["default"]
        }
      },
      "consul": {
        "address": "http://consul-host:8500",
        "datacenter": "dc1",
        "publisher_max_routines": 500
      }
    }

All available configuration options for Consul are:

.. _option_consul_addr_cfg:

  * ``address``: Equivalent to :ref:`--consul_address <option_consul_addr_cmd>` command-line flag.

.. _option_consul_token_cfg:

  * ``token``: Equivalent to :ref:`--consul_token <option_consul_token_cmd>` command-line flag.

.. _option_consul_dc_cfg:

  * ``datacenter``: Equivalent to :ref:`--consul_datacenter <option_consul_dc_cmd>` command-line flag.

.. _option_consul_key_cfg:

  * ``key_file``: Equivalent to :ref:`--consul_key_file <option_consul_key_cmd>` command-line flag.

.. _option_consul_cert_cfg:

  * ``cert_file``: Equivalent to :ref:`--consul_cert_file <option_consul_cert_cmd>` command-line flag.

.. _option_consul_ca_cert_cfg:

  * ``ca_cert``: Equivalent to :ref:`--consul_ca_cert <option_consul_ca_cert_cmd>` command-line flag.

.. _option_consul_ca_path_cfg:

  * ``ca_path``: Equivalent to :ref:`--consul_ca_path <option_consul_ca_path_cmd>` command-line flag.

.. _option_consul_ssl_cfg:

  * ``ssl``: Equivalent to :ref:`--consul_ssl <option_consul_ssl_cmd>` command-line flag.

.. _option_consul_ssl_verify_cfg:

  * ``ssl_verify``: Equivalent to :ref:`--consul_ssl_verify <option_consul_ssl_verify_cmd>` command-line flag.

.. _option_pub_routines_cfg:

  * ``publisher_max_routines``: Equivalent to :ref:`--consul_publisher_max_routines <option_pub_routines_cmd>` command-line flag.

.. _yorc_config_file_telemetry_section:

Telemetry configuration
~~~~~~~~~~~~~~~~~~~~~~~

Telemetry configuration can only be done via the configuration file.
By default telemetry data are only stored in memory.
See :ref:`yorc_telemetry_section` for more information about telemetry.

Below is an example of configuration file with telemetry metrics forwarded to a ``Statsd`` instance and with a ``Prometheus`` HTTP endpoint exposed.

.. code-block:: JSON
    
    {
      "resources_prefix": "yorc1-",
      "infrastructures": {
        "openstack": {
          "auth_url": "http://your-openstack:5000/v2.0",
          "tenant_name": "your-tenant",
          "user_name": "os-user",
          "password": "os-password",
          "private_network_name": "default-private-network",
          "default_security_groups": ["default"]
        }
      },
      "telemetry": {
        "statsd_address": "127.0.0.1:8125",
        "expose_prometheus_endpoint": true  
      }
    }

All available configuration options for telemetry are:

.. _option_telemetry_srvname_cfg:

  * ``service_name``: Metrics keys prefix, defaults to ``yorc``.

.. _option_telemetry_disHostName_cfg:

  * ``disable_hostname``: Specifies if gauge values should not be prefixed with the local hostname. Defaults to ``false``.

.. _option_telemetry_disRuntimeMetrics_cfg:

  * ``disable_go_runtime_metrics``: Specifies Go runtime metrics (goroutines, memory, ...) should not be published. Defaults to ``false``.

.. _option_telemetry_statsd_cfg:

  * ``statsd_address``: Specify the address (in form <address>:<port>) of a statsd server to forward metrics data to. 


.. _option_telemetry_statsite_cfg:

  * ``statsite_address``: Specify the address (in form <address>:<port>) of a statsite server to forward metrics data to.

.. _option_telemetry_prom_cfg:

  * ``expose_prometheus_endpoint``: Specify if an HTTP Prometheus endpoint should be exposed allowing Prometheus to scrape metrics.

.. _yorc_config_file_deprecated_section:

Deprecated configuration options
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. deprecated:: 3.0.0
.. _option_deprecated_ansible_ssh_cfg:

  * ``ansible_use_openssh``: Equivalent to :ref:`--ansible_use_openssh <option_ansible_ssh_cmd>` command-line flag.

.. _option_deprecated_ansible_debug_cfg:

  * ``ansible_debug``: Equivalent to :ref:`--ansible_debug <option_ansible_debug_cmd>` command-line flag.

.. _option_deprecated_ansible_connection_retries_cfg:

  * ``ansible_connection_retries``: Equivalent to :ref:`--ansible_connection_retries <option_ansible_connection_retries_cmd>` command-line flag.

.. _option_deprecated_operation_remote_base_dir_cfg:

  * ``operation_remote_base_dir``: Equivalent to :ref:`--operation_remote_base_dir <option_operation_remote_base_dir_cmd>` command-line flag.

.. _option_deprecated_keep_remote_path_cfg:

  * ``keep_operation_remote_path``: Equivalent to :ref:`--keep_operation_remote_path <option_keep_remote_path_cmd>` command-line flag.

.. _option_deprecated_consul_addr_cfg:

  * ``consul_address``: Equivalent to :ref:`--consul_address <option_consul_addr_cmd>` command-line flag.

.. _option_deprecated_consul_token_cfg:

  * ``consul_token``: Equivalent to :ref:`--consul_token <option_consul_token_cmd>` command-line flag.

.. _option_deprecated_consul_dc_cfg:

  * ``consul_datacenter``: Equivalent to :ref:`--consul_datacenter <option_consul_dc_cmd>` command-line flag.

.. _option_deprecated_consul_key_cfg:

  * ``consul_key_file``: Equivalent to :ref:`--consul_key_file <option_consul_key_cmd>` command-line flag.

.. _option_deprecated_consul_cert_cfg:

  * ``consul_cert_file``: Equivalent to :ref:`--consul_cert_file <option_consul_cert_cmd>` command-line flag.

.. _option_deprecated_consul_ca_cert_cfg:

  * ``consul_ca_cert``: Equivalent to :ref:`--consul_ca_cert <option_consul_ca_cert_cmd>` command-line flag.

.. _option_deprecated_consul_ca_path_cfg:

  * ``consul_ca_path``: Equivalent to :ref:`--consul_ca_path <option_consul_ca_path_cmd>` command-line flag.

.. _option_deprecated_consul_ssl_cfg:

  * ``consul_ssl``: Equivalent to :ref:`--consul_ssl <option_consul_ssl_cmd>` command-line flag.

.. _option_deprecated_consul_ssl_verify_cfg:

  * ``consul_ssl_verify``: Equivalent to :ref:`--consul_ssl_verify <option_consul_ssl_verify_cmd>` command-line flag.

.. _option_deprecated_pub_routines_cfg:

  * ``consul_publisher_max_routines``: Equivalent to :ref:`--consul_publisher_max_routines <option_pub_routines_cmd>` command-line flag.

Environment variables
---------------------

.. _option_ansible_ssh_env:

  * ``YORC_ANSIBLE_USE_OPENSSH``: Equivalent to :ref:`--ansible_use_openssh <option_ansible_ssh_cmd>` command-line flag.

.. _option_ansible_debug_env:

  * ``YORC_ANSIBLE_DEBUG``: Equivalent to :ref:`--ansible_debug <option_ansible_debug_cmd>` command-line flag.

.. _option_ansible_connection_retries_env:

  * ``YORC_ANSIBLE_CONNECTION_RETRIES``: Equivalent to :ref:`--ansible_connection_retries <option_ansible_connection_retries_cmd>` command-line flag.

.. _option_operation_remote_base_dir_env:

  * ``YORC_OPERATION_REMOTE_BASE_DIR``: Equivalent to :ref:`--operation_remote_base_dir <option_operation_remote_base_dir_cmd>` command-line flag.

.. _option_consul_addr_env:

  * ``YORC_CONSUL_ADDRESS``: Equivalent to :ref:`--consul_address <option_consul_addr_cmd>` command-line flag.

.. _option_consul_token_env:

  * ``YORC_CONSUL_TOKEN``: Equivalent to :ref:`--consul_token <option_consul_token_cmd>` command-line flag.

.. _option_consul_dc_env:

  * ``YORC_CONSUL_DATACENTER``: Equivalent to :ref:`--consul_datacenter <option_consul_dc_cmd>` command-line flag.

.. _option_consul_key_file_env:

  * ``YORC_CONSUL_KEY_FILE``: Equivalent to :ref:`--consul_key_file <option_consul_key_cmd>` command-line flag.

.. _option_consul_cert_file_env:

  * ``YORC_CONSUL_CERT_FILE``: Equivalent to :ref:`--consul_cert_file <option_consul_cert_cmd>` command-line flag.

.. _option_consul_ca_cert_env:

  * ``YORC_CONSUL_CA_CERT``: Equivalent to :ref:`--consul_ca_cert <option_consul_ca_cert_cmd>` command-line flag.

.. _option_consul_ca_path_env:

  * ``YORC_CONSUL_CA_PATH``: Equivalent to :ref:`--consul_ca_path <option_consul_ca_path_cmd>` command-line flag.

.. _option_consul_ssl_env:

  * ``YORC_CONSUL_SSL``: Equivalent to :ref:`--consul_ssl <option_consul_ssl_cmd>` command-line flag.

.. _option_consul_ssl_verify_env:

  * ``YORC_CONSUL_SSL_VERIFY``: Equivalent to :ref:`--consul_ssl_verify <option_consul_ssl_verify_cmd>` command-line flag.

.. _option_pub_routines_env:

  * ``YORC_CONSUL_PUBLISHER_MAX_ROUTINES``: Equivalent to :ref:`--consul_publisher_max_routines <option_pub_routines_cmd>` command-line flag.

.. _option_shut_timeout_env:

  * ``YORC_SERVER_GRACEFUL_SHUTDOWN_TIMEOUT``: Equivalent to :ref:`--graceful_shutdown_timeout <option_shut_timeout_cmd>` command-line flag.

.. _option_wf_step_termination_timeout_env:

  * ``YORC_WF_STEP_GRACEFUL_TERMINATION_TIMEOUT``: Equivalent to :ref:`--wf_step_graceful_termination_timeout <option_wf_step_termination_timeout_cmd>` command-line flag.

.. _option_http_addr_env:

  * ``YORC_HTTP_ADDRESS``: Equivalent to :ref:`--http_address <option_http_addr_cmd>` command-line flag.

.. _option_http_port_env:

  * ``YORC_HTTP_PORT``: Equivalent to :ref:`--http_port <option_http_port_cmd>` command-line flag.

.. _option_keep_remote_path_env:

  * ``YORC_KEEP_OPERATION_REMOTE_PATH``: Equivalent to :ref:`--keep_operation_remote_path <option_keep_remote_path_cmd>` command-line flag.

.. _option_keyfile_env:

  * ``YORC_KEY_FILE``: Equivalent to :ref:`--key_file <option_keyfile_cmd>` command-line flag.

.. _option_certfile_env:

  * ``YORC_CERT_FILE``: Equivalent to :ref:`--cert_file <option_certfile_cmd>` command-line flag.

.. _option_sslverify_env:

  * ``YORC_CERT_FILE``: Equivalent to :ref:`--ssl_verify <option_sslverify_cmd>` command-line flag.

.. _option_cafile_env:

  * ``YORC_CA_FILE``: Equivalent to :ref:`--ca_file <option_cafile_cmd>` command-line flag.

.. _option_plugindir_env:

  * ``YORC_PLUGINS_DIRECTORY``: Equivalent to :ref:`--plugins_directory <option_pluginsdir_cmd>` command-line flag.

.. _option_resources_prefix_env:

  * ``YORC_RESOURCES_PREFIX``: Equivalent to :ref:`--resources_prefix <option_resources_prefix_cmd>` command-line flag.

.. _option_workers_env:

  * ``YORC_WORKERS_NUMBER``: Equivalent to :ref:`--workers_number <option_workers_cmd>` command-line flag.

.. _option_workdir_env: 

  * ``YORC_WORKING_DIRECTORY``: Equivalent to :ref:`--working_directory <option_workdir_cmd>` command-line flag.

.. _option_server_id_env:

  * ``YORC_SERVER_ID``: Equivalent to :ref:`--server_id <option_server_id_cmd>` command-line flag.

.. _option_log_env: 

  * ``YORC_LOG``: If set to ``1`` or ``DEBUG``, enables debug logging for Yorc.

.. _option_aws_access_key:

  * ``YORC_INFRA_AWS_ACCESS_KEY``: The AWS access key credential.

.. _option_aws_secret_key:

  * ``YORC_INFRA_AWS_SECRET_KEY``: The AWS secret key credential.
 

Infrastructures configuration
-----------------------------

Due to the pluggable nature of infrastructures support in Yorc their configuration differ from other configurable options.
An infrastructure configuration option could be specified by either a its configuration placeholder in the configuration file, a command line flag
or an environment variable.

The general principle is for a configurable option ``option_1`` for infrastructure ``infra1`` it should be specified in the configuration file as following:

.. code-block:: JSON
    
    {
      "infrastructures": {
        "infra1": {
          "option_1": "value"
        }
      }
    }
  
Similarly a command line flag with the name ``--infrastructure_infra1_option_1`` and an environment variable with the name ``YORC_INFRA_INFRA1_OPTION_1`` will be
automatically supported and recognized. The default order of precedence apply here.

Builtin infrastructures configuration
-------------------------------------

.. _option_infra_os: 

OpenStack
~~~~~~~~~

OpenStack infrastructure key name is ``openstack`` in lower case.

.. 
   MAG - According to:
   https://github.com/sphinx-doc/sphinx/issues/3043
   http://www.sphinx-doc.org/en/stable/markup/misc.html#tables
.. tabularcolumns:: |p{0.35\textwidth}|p{0.30\textwidth}|p{0.05\textwidth}|p{0.15\textwidth}|p{0.10\textwidth}|

+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
|            Option Name            |                                                     Description                                                     | Data Type |                      Required                      |    Default    |
|                                   |                                                                                                                     |           |                                                    |               |
+===================================+=====================================================================================================================+===========+====================================================+===============+
| ``auth_url``                      | Specify the authentication url for OpenStack (should be the Keystone endpoint ie: http://your-openstack:5000/v2.0). | string    | yes                                                |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``tenant_id``                     | Specify the OpenStack tenant id to use.                                                                             | string    | Either this or ``tenant_name`` should be provided. |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``tenant_name``                   | Specify the OpenStack tenant name to use.                                                                           | string    | Either this or ``tenant_id`` should be provided.   |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``user_name``                     | Specify the OpenStack user name to use.                                                                             | string    | yes                                                |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``password``                      | Specify the OpenStack password to use.                                                                              | string    | yes                                                |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``region``                        | Specify the OpenStack region to use                                                                                 | string    | no                                                 | ``RegionOne`` |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``private_network_name``          | Specify the name of private network to use as primary adminstration network between Yorc and Compute                | string    | Required to use the ``PRIVATE`` keyword for TOSCA  |               |
|                                   | instances. It should be a private network accessible by this instance of Yorc.                                      |           | admin networks                                     |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``provisioning_over_fip_allowed`` | This allows to perform the provisioning of a Compute over the associated floating IP if it exists. This is useful   | boolean   | no                                                 | ``false``     |
|                                   | when Yorc is not deployed on the same private network than the provisioned Compute.                                 |           |                                                    |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``default_security_groups``       | Default security groups to be used when creating a Compute instance. It should be a comma-separated list of         | list of   | no                                                 |               |
|                                   | security group names                                                                                                | strings   |                                                    |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``insecure``                      | Trust self-signed SSL certificates                                                                                  | boolean   | no                                                 | ``false``     |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``cacert_file``                   | Specify a custom CA certificate when communicating over SSL. You can specify either a path to the file or the       | string    | no                                                 |               |
|                                   | contents of the certificate                                                                                         |           |                                                    |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``cert``                          | Specify client certificate file for SSL client authentication. You can specify either a path to the file or         | string    | no                                                 |               |
|                                   | the contents of the certificate                                                                                     |           |                                                    |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``key``                           | Specify client private key file for SSL client authentication. You can specify either a path to the file or         | string    | no                                                 |               |
|                                   | the contents of the key                                                                                             |           |                                                    |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+


.. _option_infra_kubernetes: 

Kubernetes
~~~~~~~~~~

Kubernetes infrastructure key name is ``kubernetes`` in lower case.

.. 
   MAG - According to:
   https://github.com/sphinx-doc/sphinx/issues/3043
   http://www.sphinx-doc.org/en/stable/markup/misc.html#tables
.. tabularcolumns:: |l|L|L|L|L|

+----------------+---------------------------------------------------------------------------------+-----------+----------+---------+
|  Option Name   |                                   Description                                   | Data Type | Required | Default |
|                |                                                                                 |           |          |         |
+================+=================================================================================+===========+==========+=========+
| ``master_url`` | URL of the HTTP API of Kubernetes is exposed. Format: ``https://<host>:<port>`` | string    | yes      |         |
+----------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``ca_file``    | Path to a trusted root certificates for server                                  | string    | no       |         |
+----------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``cert_file``  | Path to the TLS client certificate used for authentication                      | string    | no       |         |
+----------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``key_file``   | Path to the TLS client key used for authentication                              | string    | no       |         |
+----------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``insecure``   | Server should be accessed without verifying the TLS certificate (testing only)  | boolean   | no       |         |
+----------------+---------------------------------------------------------------------------------+-----------+----------+---------+

.. _option_infra_google:

Google Cloud Platform
~~~~~~~~~~~~~~~~~~~~~

Google Cloud Platform infrastructure key name is ``google`` in lower case.

+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+
|  Option Name                |              Description                     | Data Type | Required | Default                                |
|                             |                                              |           |          |                                        |
+=============================+==============================================+===========+==========+========================================+
| ``project``                 | ID of the project to apply any resources to  | string    | yes      |                                        |
+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+
| ``application_credentials`` | Path of file containing credentials*         | string    | no       | Google Application Default Credentials |
+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+
| ``region``                  | The region to operate under                  | string    | no       |                                        |
+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+

*``application_credentials`` is the path (accessible to Yorc server) of a file containing service account private keys in JSON format.
This file can be downloaded from the Google Cloud Console at  `Google Cloud service account file <https://console.cloud.google.com/apis/credentials/serviceaccountkey>`_.

If no file is specified, the orchestrator will fall back to using the `Google Application Default Credentials <https://cloud.google.com/docs/authentication/production>`_ if any.

.. _option_infra_aws:

AWS
~~~~~~~~~~

AWS infrastructure key name is ``aws`` in lower case.

+----------------+----------------------------------------+-----------+----------+---------+
|  Option Name   |              Description               | Data Type | Required | Default |
|                |                                        |           |          |         |
+================+========================================+===========+==========+=========+
| ``access_key`` | Specify the AWS access key credential. | string    | yes      |         |
+----------------+----------------------------------------+-----------+----------+---------+
| ``secret_key`` | Specify the AWS secret key credential. | string    | yes      |         |
+----------------+----------------------------------------+-----------+----------+---------+
| ``region``     | Specify the AWS region to use.         | string    | yes      |         |
+----------------+----------------------------------------+-----------+----------+---------+

.. _option_infra_slurm:

Slurm
~~~~~~~~~~

Slurm infrastructure key name is ``slurm`` in lower case.

+----------------------+------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
|     Option Name      |                          Description                             | Data Type |                     Required                      | Default |
|                      |                                                                  |           |                                                   |         |
+======================+==================================================================+===========+===================================================+=========+
| ``user_name``        | SSH Username to be used to connect to the Slurm Client's node    | string    | yes                                               |         |
+----------------------+------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``password``         | SSH Password to be used to connect to the Slurm Client's node    | string    | Either this or ``private_key`` should be provided |         |
+----------------------+------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``private_key``      | SSH Private key to be used to connect to the Slurm Client's node | string    | Either this or ``password`` should be provided    |         |
+----------------------+------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``url``              | IP address of the Slurm Client's node                            | string    | yes                                               |         |
+----------------------+------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``port``             | SSH Port to be used to connect to the Slurm Client's node        | string    | yes                                               |         |
+----------------------+------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``default_job_name`` | Default name for the job allocation.                             | string    | no                                                |         |
+----------------------+------------------------------------------------------------------+-----------+---------------------------------------------------+---------+


Vault configuration
-------------------

Due to the pluggable nature of vaults support in Yorc their configuration differ from other configurable options.
A vault configuration option could be specified by either its configuration placeholder in the configuration file, a command line flag
or an environment variable.

The general principle is for a configurable option ``option_1`` it should be specified in the configuration file as following:

.. code-block:: JSON
    
    {
      "vault": {
        "type": "vault_implementation",
        "option_1": "value"
      }
    }
  
Similarly a command line flag with the name ``--vault_option_1`` and an environment variable with the name ``YORC_VAULT_OPTION_1`` will be
automatically supported and recognized. The default order of precedence apply here.

``type`` is the only mandatory option for all vaults configurations, it allows to select the vault implementation by specifying it's ID. If the
``type`` option is not present either in the config file, as a command line flag or as an environment variable, Vault configuration will be ignored.

The integration with a Vault is totally optional and this configuration part may be leave empty.  

Builtin Vaults configuration
----------------------------

.. _option_hashivault: 

HashiCorp's Vault
~~~~~~~~~~~~~~~~~

This is the only builtin supported Vault implementation. 
Implementation ID to use with the vault type configuration parameter is ``hashicorp``.


Bellow are recognized configuration options for Vault:

.. 
   MAG - According to:
   https://github.com/sphinx-doc/sphinx/issues/3043
   http://www.sphinx-doc.org/en/stable/markup/misc.html#tables
.. tabularcolumns:: |l|L|l|l|l|

+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
|     Option Name     |                                                            Description                                                            | Data Type | Required |  Default  |
|                     |                                                                                                                                   |           |          |           |
+=====================+===================================================================================================================================+===========+==========+===========+
| ``address``         | Address is the address of the Vault server. This should be a complete URL such as "https://vault.example.com".                    | string    | yes      |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``max_retries``     | MaxRetries controls the maximum number of times to retry when a 5xx error occurs. Set to 0 or less to disable                     | integer   | no       | ``0``     |
|                     | retrying.                                                                                                                         |           |          |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``timeout``         | Timeout is for setting custom timeout parameter in the HttpClient.                                                                | string    | no       |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``ca_cert``         | CACert is the path to a PEM-encoded CA cert file to use to verify the Vault server SSL certificate.                               | string    | no       |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``ca_path``         | CAPath is the path to a directory of PEM-encoded CA cert files to verify the Vault server SSL certificate.                        | string    | no       |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``client_cert``     | ClientCert is the path to the certificate for Vault communication.                                                                | string    | no       |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``client_key``      | ClientKey is the path to the private key for Vault communication                                                                  | string    | no       |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``tls_server_name`` | TLSServerName, if set, is used to set the SNI host when connecting via TLS.                                                       | string    | no       |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``tls_skip_verify`` | Disables SSL verification                                                                                                         | boolean   | no       | ``false`` |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+
| ``token``           | Specifies the access token to use to connect to vault.  This is highly discouraged to this option in the                          | string    | no       |           |
|                     | configuration file as the token is a sensitive data and should not be written on disk. Prefer the associated environment variable |           |          |           |
+---------------------+-----------------------------------------------------------------------------------------------------------------------------------+-----------+----------+-----------+

.. _yorc_config_client_section:

Yorc Client CLI Configuration
=============================

This section is dedicated to the CLI part of yorc that covers everything except the server configuration detailed
above. It focus on configuration options commons to all the commands. Sub commands may have additional options please use the cli *help* command to see them.

Just like for its server part Yorc Client CLI has various configuration options that could be specified either by command-line flags, configuration file or environment variables.

If an option is specified several times using flags, environment and config file, command-line flag will have the precedence then the environment variable and finally the value defined in the configuration file. 

Command-line options
--------------------


.. _option_client_ca_file_cmd:

  * ``--ca_file``: This provides a file path to a PEM-encoded certificate authority. This implies the use of HTTPS to connect to the Yorc REST API.

.. _option_client_ca_path_cmd:

  * ``--ca_path``: Path to a directory of PEM-encoded certificates authorities. This implies the use of HTTPS to connect to the Yorc REST API.

.. _option_client_cert_file_cmd:

  * ``--cert_file``: File path to a PEM-encoded client certificate used to authenticate to the Yorc API. This must be provided along with key-file. If one of key-file or cert-file is not provided then SSL authentication is disabled. If both cert-file and key-file are provided this implies the use of HTTPS to connect to the Yorc REST API.

.. _option_client_config_cmd:

  * ``-c`` or ``--config``: config file (default is /etc/yorc/yorc-client.[json|yaml])

.. _option_client_key_file_cmd:

  * ``--key_file``: File path to a PEM-encoded client private key used to authenticate to the Yorc API. This must be provided along with cert-file. If one of key-file or cert-file is not provided then SSL authentication is disabled. If both cert-file and key-file are provided this implies the use of HTTPS to connect to the Yorc REST API.

.. _option_client_skip_tls_verify_cmd:

  * ``--skip_tls_verify``: Controls whether a client verifies the server's certificate chain and host name. If set to true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing. This implies the use of HTTPS to connect to the Yorc REST API.

.. _option_client_skip_tls_verify_cmd:

  * ``-s`` or ``--ssl_enabled``: Use HTTPS to connect to the Yorc REST API. This is automatically implied if one of ``--ca_file``, ``--ca_path``, ``--cert_file``, ``--key_file`` or ``--skip_tls_verify`` is provided.

.. _option_client_yorc_api_cmd:

  * ``--yorc_api``: specify the host and port used to join the Yorc' REST API (default "localhost:8800")

Configuration files
-------------------

Configuration files are either JSON or YAML formatted as a single object containing the following configuration options. 
By default Yorc will look for a file named yorc-client.json or yorc-client.yaml in ``/etc/yorc`` directory then if not found in the current directory. 
The :ref:`--config <option_client_config_cmd>` command line flag allows to specify an alternative configuration file.

.. _option_client_ca_file_cfg:

  * ``ca_file``: Equivalent to :ref:`--ca_file <option_client_ca_file_cmd>` command-line flag.

.. _option_client_ca_path_cfg:

  * ``ca_path``: Equivalent to :ref:`--ca_path <option_client_ca_path_cmd>` command-line flag.

.. _option_client_cert_file_cfg:

  * ``cert_file``: Equivalent to :ref:`--cert_file <option_client_cert_file_cmd>` command-line flag.

.. _option_client_key_file_cfg:

  * ``key_file``: Equivalent to :ref:`--key_file <option_client_key_file_cmd>` command-line flag.

.. _option_client_skip_tls_verify_cfg:

  * ``skip_tls_verify``: Equivalent to :ref:`--skip_tls_verify <option_client_skip_tls_verify_cmd>` command-line flag.

.. _option_client_skip_tls_verify_cfg:

  * ``ssl_enabled``: Equivalent to :ref:`--skip_tls_verify <option_client_skip_tls_verify_cmd>` command-line flag.

.. _option_client_yorc_api_cfg:

  * ``yorc_api``: Equivalent to :ref:`--yorc_api <option_client_yorc_api_cmd>` command-line flag.

Environment variables
---------------------

.. _option_client_ca_file_env:

  * ``YORC_CA_FILE``: Equivalent to :ref:`--ca_file <option_client_ca_file_cmd>` command-line flag.

.. _option_client_ca_path_env:

  * ``YORC_CA_PATH``: Equivalent to :ref:`--ca_path <option_client_ca_path_cmd>` command-line flag.

.. _option_client_cert_file_env:

  * ``YORC_CERT_FILE``: Equivalent to :ref:`--cert_file <option_client_cert_file_cmd>` command-line flag.

.. _option_client_key_file_env:

  * ``YORC_KEY_FILE``: Equivalent to :ref:`--key_file <option_client_key_file_cmd>` command-line flag.

.. _option_client_skip_tls_verify_env:

  * ``YORC_SKIP_TLS_VERIFY``: Equivalent to :ref:`--skip_tls_verify <option_client_skip_tls_verify_cmd>` command-line flag.

.. _option_client_skip_tls_verify_env:

  * ``YORC_SSL_ENABLED``: Equivalent to :ref:`--skip_tls_verify <option_client_skip_tls_verify_cmd>` command-line flag.

.. _option_client_yorc_api_env:

  * ``YORC_API``: Equivalent to :ref:`--yorc_api <option_client_yorc_api_cmd>` command-line flag.
