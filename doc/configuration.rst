..
   Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
   ---

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

.. _option_ansible_cache_facts_cmd:

  * ``--ansible_cache_facts``: If set to true, caches Ansible facts (values fetched on remote hosts about network/hardware/OS/virtualization configuration) so that these facts are not recomputed each time a new operation is a run for a given deployment (false by default: no caching).

.. _option_ansible_archive_artifacts_cmd:

  * ``--ansible_archive_artifacts``: If set to true, archives operation bash/python scripts locally, copies this archive and unarchives it on remote hosts (requires tar to be installed on remote hosts), to avoid multiple time consuming remote copy operations of individual scripts (false by default: no archive).

.. _option_ansible_job_monitoring_time_interval_cmd:

  * ``--ansible_job_monitoring_time_interval``: Default duration for monitoring time interval for jobs handled by Ansible (defaults to 15s).

.. _option_ansible_keep_generated_recipes_cmd:

  * ``--ansible_keep_generated_recipes``: If set to true, generated Ansible recipes on Yorc server are not deleted. (false by default: generated recipes are deleted).

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

.. _option_consul_tls_handshake_timeout_cmd:

  * ``--consul_tls_handshake_timeout``: Maximum duration to wait for a TLS handshake with Consul, the default is ``50s``.

.. _option_terraform_plugins_dir_cmd:

  * ``--terraform_plugins_dir``: Specify the directory where to find Terraform pre-installed providers plugins. If not specified, required plugins will be downloaded during deployment. See https://www.terraform.io/guides/running-terraform-in-automation.html#pre-installed-plugins for more information.

.. _option_terraform_aws_plugin_version_constraint_cmd:

  * ``--terraform_aws_plugin_version_constraint``: Specify the Terraform AWS plugin version constraint. Default one compatible with our source code is ``"~> 1.36"``. If you choose another, it's at your own risk. See https://www.terraform.io/docs/configuration/providers.html#provider-versions for more information.

.. _option_terraform_consul_plugin_version_constraint_cmd:

  * ``--terraform_consul_plugin_version_constraint``: Specify the Terraform Consul plugin version constraint. Default one compatible with our source code is ``"~> 2.1"``. If you choose another, it's at your own risk. See https://www.terraform.io/docs/configuration/providers.html#provider-versions for more information.

.. _option_terraform_google_plugin_version_constraint_cmd:

  * ``--terraform_google_plugin_version_constraint``: Specify the Terraform Google plugin version constraint. Default one compatible with our source code is ``"~> 1.18"``. If you choose another, it's at your own risk. See https://www.terraform.io/docs/configuration/providers.html#provider-versions for more information.

.. _option_terraform_openstack_plugin_version_constraint_cmd:

  * ``--terraform_openstack_plugin_version_constraint``: Specify the Terraform OpenStack plugin version constraint. Default one compatible with our source code is ``"~> 1.9"``. If you choose another, it's at your own risk. See https://www.terraform.io/docs/configuration/providers.html#provider-versions for more information.

.. _option_terraform_keep_generated_files_cmd:

  * ``--terraform_keep_generated_files``: If set to true, generated Terraform infrastructures files on Yorc server are not deleted. (false by default: generated files are deleted).

.. _option_pub_routines_cmd:

  * ``--consul_publisher_max_routines``: Maximum number of parallelism used to store key/values in Consul. If you increase the default value you may need to tweak the ulimit max open files. If set to 0 or less the default value (500) will be used.

.. _option_shut_timeout_cmd:

  * ``--graceful_shutdown_timeout``: Timeout to wait for a graceful shutdown of the Yorc server. After this delay the server immediately exits. The default is ``5m``.

.. _option_wf_step_termination_timeout_cmd:

  * ``--wf_step_graceful_termination_timeout``: Timeout to wait for a graceful termination of a workflow step during concurrent workflow step failure. After this delay the step is set on error. The default is ``2m``.

.. _option_purged_deployments_eviction_timeout_cmd:

  * ``--purged_deployments_eviction_timeout``: When a deployment is purged an event is kept to let a chance to external systems to detect it via the events API, this timeout controls the retention time of such events. The default is ``30m``.

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

.. _option_locations_cmd:

  * ``--locations_file_path``: File path to locations configuration. This configuration is taken in account for the first time the server starts.

.. _option_resources_prefix_cmd:

  * ``--resources_prefix``: Specify a prefix that will be used for names when creating resources such as Compute instances or volumes. Defaults to ``yorc-``.

.. _option_tasks_dispatcher_long_polling_wait_time_cmd:

  * ``--tasks_dispatcher_long_polling_wait_time``: Wait time (Golang duration format) when long polling for executions tasks to dispatch to workers. If not set the default value of `1m` will be used.

.. _option_tasks_dispatcher_lock_wait_time_cmd:

  * ``--tasks_dispatcher_lock_wait_time``: Wait time (Golang duration format) for acquiring a lock for an execution task. If not set the default value of `50ms` will be used.

.. _option_tasks_dispatcher_metrics_refresh_time_cmd:

  * ``--tasks_dispatcher_metrics_refresh_time``: Refresh time (Golang duration format) for the tasks dispatcher metrics. If not set the default value of `5m` will be used.

.. _option_workers_cmd:

  * ``--workers_number``: Yorc instances use a pool of workers to handle deployment tasks. This option defines the size of this pool. If not set the default value of `30` will be used.

.. _option_workdir_cmd:

  * ``--working_directory`` or ``-w``: Specify an alternative working directory for Yorc. The default is to use a directory named *work* in the current directory.

.. _option_server_id_cmd:

  * ``--server_id``: Specify the server ID used to identify the server node in a cluster. The default is the hostname.

.. _option_disable_ssh_agent_cmd:

  * ``--disable_ssh_agent``: Allow disabling ssh-agent use for SSH authentication on provisioned computes. Default is false. If true, compute credentials must provide a path to a private key file instead of key content.

.. _option_upgrades_concurrency_cmd:

  * ``--concurrency_limit_for_upgrades``: Limit of concurrency used in Upgrade processes. If not set the default value of `1000` will be used.

.. _option_ssh_connection_timeout_cmd:

  * ``--ssh_connection_timeout``: Timeout to establish SSH connection from Yorc SSH client, especially used for Slurm and HostsPool locations. If not set the default value of `10 s` will be used.

.. _option_ssh_connection_retry_backoff_cmd:

  * ``--ssh_connection_retry_backoff``: Backoff duration before retrying an ssh connection. This may be superseded by a location attribute if supported. (default 1s)

.. _option_ssh_connection_max_retries_cmd:

  * ``--ssh_connection_max_retries``: Maximum number of retries (attempts are retries + 1) before giving-up to connect. This may be superseded by a location attribute if supported. (default 3)


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
      "locations_file_path": "path-to-locations-yaml-or-json-config"
    }


Below is an example of configuration file with TLS enabled.

.. code-block:: JSON

    {
      "resources_prefix": "yorc1-",
      "key_file": "/etc/pki/tls/private/yorc.key",
      "cert_file": "/etc/pki/tls/certs/yorc.crt",
      "locations_file_path": "path-to-locations-yaml-or-json-config"
    }

.. _option_shut_timeout_cfg:

  * ``server_graceful_shutdown_timeout``: Equivalent to :ref:`--graceful_shutdown_timeout <option_shut_timeout_cmd>` command-line flag.

.. _option_wf_step_termination_timeout_cfg:

  * ``wf_step_graceful_termination_timeout``: Equivalent to :ref:`--wf_step_graceful_termination_timeout <option_wf_step_termination_timeout_cmd>` command-line flag.

.. _option_purged_deployments_eviction_timeout_cfg:

  * ``purged_deployments_eviction_timeout``: Equivalent to :ref:`--purged_deployments_eviction_timeout <option_purged_deployments_eviction_timeout_cmd>` command-line flag.

.. _option_http_addr_cfg:

  * ``http_address``: Equivalent to :ref:`--http_address <option_http_addr_cmd>` command-line flag.

.. _option_http_port_cfg:

  * ``http_port``: Equivalent to :ref:`--http_port <option_http_port_cmd>` command-line flag.

.. _option_keyfile_cfg:

  * ``key_file``: Equivalent to :ref:`--key_file <option_keyfile_cmd>` command-line flag.

.. _option_certfile_cfg:

  * ``cert_file``: Equivalent to :ref:`--cert_file <option_certfile_cmd>` command-line flag.

.. _option_sslverify_cfg:

  * ``ssl_verify``: Equivalent to :ref:`--ssl_verify <option_ssl_verify_cmd>` command-line flag.

.. _option_ca_file_cfg:

  * ``ca_file``: Equivalent to :ref:`--ca_file <option_ca_file_cmd>` command-line flag.

.. _option_plugindir_cfg:

  * ``plugins_directory``: Equivalent to :ref:`--plugins_directory <option_pluginsdir_cmd>` command-line flag.

.. _option_resources_prefix_cfg:

  * ``resources_prefix``: Equivalent to :ref:`--resources_prefix <option_resources_prefix_cmd>` command-line flag.

.. _option_locations_cfg:

  * ``locations_file_path``: Equivalent to :ref:`--locations_file_path <option_locations_cmd>` command-line flag.

.. _option_workers_cfg:

  * ``workers_number``: Equivalent to :ref:`--workers_number <option_workers_cmd>` command-line flag.

.. _option_workdir_cfg:

  * ``working_directory``: Equivalent to :ref:`--working_directory <option_workdir_cmd>` command-line flag.

.. _option_server_id_cfg:

  * ``server_id``: Equivalent to :ref:`--server_id <option_server_id_cmd>` command-line flag.

.. _option_disable_ssh_agent_cfg:

  * ``disable_ssh_agent``: Equivalent to :ref:`--disable_ssh_agent <option_disable_ssh_agent_cmd>` command-line flag.

.. _option_upgrades_concurrency_cfg:

  * ``concurrency_limit_for_upgrades``: Equivalent to :ref:`--concurrency_limit_for_upgrades <option_upgrades_concurrency_cmd>` command-line flag.

.. _option_ssh_connection_timeout_cfg:

  * ``ssh_connection_timeout``: Equivalent to :ref:`--ssh_connection_timeout <option_ssh_connection_timeout_cmd>` command-line flag.

.. _option_ssh_connection_retry_backoff_cfg:

  * ``ssh_ssh_connection_retry_backoff``: Equivalent to :ref:`--ssh_ssh_connection_retry_backoff <option_ssh_connection_retry_backoff_cmd>` command-line flag.

.. _option_ssh_connection_max_retries_cfg:

  * ``ssh_connection_max_retries``: Equivalent to :ref:`--ssh_connection_max_retries <option_ssh_connection_max_retries_cmd>` command-line flag.

.. _yorc_config_file_ansible_section:

Ansible configuration
~~~~~~~~~~~~~~~~~~~~~

Below is an example of configuration file with Ansible configuration options.

.. code-block:: JSON

    {
      "resources_prefix": "yorc1-",
      "locations_file_path": "path-to-locations-yaml-or-json-config",
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
        },
        "config": {
          "defaults": {
            "display_skipped_hosts": "False",
            "special_context_filesystems": "nfs,vboxsf,fuse,ramfs,myspecialfs",
            "timeout": "60"
          }
        },
        "inventory":{
          "target_hosts:vars": ["ansible_python_interpreter=/usr/bin/python3"]
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

.. _option_ansible_cache_facts_cfg:

  * ``cache_facts``: Equivalent to :ref:`--ansible_cache_facts <option_ansible_cache_facts_cmd>` command-line flag.

.. _option_ansible_archive_artifacts_cfg:

  * ``archive_artifacts``: Equivalent to :ref:`--ansible_archive_artifacts <option_ansible_archive_artifacts_cmd>` command-line flag.

.. _option_ansible_job_monitoring_time_interval_cfg:

  * ``job_monitoring_time_interval``: Equivalent to :ref:`--ansible_job_monitoring_time_interval <option_ansible_job_monitoring_time_interval_cmd>` command-line flag.

.. _option_operation_remote_base_dir_cfg:

  * ``operation_remote_base_dir``: Equivalent to :ref:`--operation_remote_base_dir <option_operation_remote_base_dir_cmd>` command-line flag.

.. _option_keep_remote_path_cfg:

  * ``keep_operation_remote_path``: Equivalent to :ref:`--keep_operation_remote_path <option_keep_remote_path_cmd>` command-line flag.

.. _option_ansible_keep_generated_recipes_cfg:

  * ``keep_generated_recipes``: Equivalent to :ref:`--ansible_keep_generated_recipes <option_ansible_keep_generated_recipes_cmd>` command-line flag.

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

      * ``config`` and ``inventory`` are complex structure allowing to configure
        Ansible behavior, these options are described in more details in next section.

.. _option_ansible_config_cfg:

Ansible config option
^^^^^^^^^^^^^^^^^^^^^

``config`` is a complex structure allowing to define `Ansible configuration settings
<https://docs.ansible.com/ansible/latest/reference_appendices/config.html>`_  if
you need a specific Ansible Configuration.

You should first provide the Ansible Configuration section (for example ``defaults``,
``ssh_connection``...).

You should then provide the list of parameters within this section, ie. what
`Ansible documentation <https://docs.ansible.com/ansible/latest/reference_appendices/config.html>`_ describes
as the ``Ini key`` within the ``Ini Section``.
Each parameter value must be provided here as a string : for a boolean parameter,
you would provide the string ``False`` or ``True`` as expected in Ansible Confiugration.
For example, it would give in Yaml:

.. code-block:: YAML

  ansible:
    config:
      defaults:
        display_skipped_hosts: "False"
        special_context_filesystems: "nfs,vboxsf,fuse,ramfs,myspecialfs"
        timeout: "60"

By default, the Orchestrator will define these Ansible Configuration settings :

  * ``host_key_checking: "False"``, to avoid host key checking by the underlying
    tools Ansible uses to connect to the host
  * ``timeout: "30"``, to set the connection timeout to 30 seconds
  * ``stdout_callback: "yaml"``, to display ansible output in yaml format
  * ``nocows: "1"``, to disable cowsay messages that can cause parsing issues in
    the Orchestrator

And when :ref:`ansible fact caching <option_ansible_cache_facts_cmd>` is enabled,
the Orchestrator adds these settings :

  * ``gathering: "smart"``, to set Ansible fact gathering to smart: each new host
    that has no facts discovered will be scanned
  * ``fact_caching: "jsonfile"``, to use a json file-based cache plugin

.. warning::
    Be careful when overriding these settings defined by default by the Orchestrator,
    as it might lead to unpredictable results.

.. _option_ansible_inventory_cfg:

Ansible inventory option
^^^^^^^^^^^^^^^^^^^^^^^^

``inventory`` is a structure allowing to configure `Ansible inventory settings
<https://docs.ansible.com/ansible/latest/user_guide/intro_inventory.html>`_  if
you need to define variables for hosts or groups.

You should first provide the Ansible Inventory group name.
You should then provide the list of parameters to define for this group, which
can be any parameter specific to your ansible playbooks, or `behavioral inventory
parameters <https://docs.ansible.com/ansible/latest/user_guide/intro_inventory.html#list-of-behavioral-inventory-parameters>`_
describing how Ansible interacts with remote hosts.

For example, for Ansible to use a given python interpreter on target hosts, you must define
the Ansible behavioral inventory parameter ``ansible_python_interpreter``
in the Ansible inventory Yorc configuration, like below in Yaml:

.. code-block:: YAML

  ansible:
    inventory:
      "target_hosts:vars":
      - ansible_python_interpreter=/usr/bin/python3

By default, the Orchestrator will define :

  * an inventory group ``target_hosts`` containing the list of remote hosts, and its
    associated variable group ``target_hosts:vars`` configuring by default these
    behavioral parameters:

    * ``ansible_ssh_common_args="-o ConnectionAttempts=20"``
    * ``ansible_python_interpreter="auto_silent"``

.. warning::
    Settings defined by the user take precedence over settings defined by the
    Orchestrator.
    Be careful when overriding these settings defined by default by the Orchestrator,
    as it might lead to unpredictable results.

Ansible performance considerations
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

As described in TOSCA :ref:`tosca_operations_implementations_section`, Yorc supports these builtin implementations for operations to execute on remote hosts :

  * Bash scripts
  * Python scripts
  * Ansible Playbooks

It is recommended to implement operations as Ansible Playbooks to get the best execution performance.

When operations are not implemented using Ansible playbooks, see the Performance section on :ref:`tosca_operations_performance_section` to improve the performance of scripts execution on remote hosts.

.. _yorc_config_file_consul_section:

Consul configuration
~~~~~~~~~~~~~~~~~~~~

Below is an example of configuration file with Consul configuration options.

.. code-block:: JSON

    {
      "resources_prefix": "yorc1-",
      "locations_file_path": "path-to-locations-yaml-or-json-config",
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

.. _option_consul_tls_handshake_timeout:

  * ``tls_handshake_timeout``: Equivalent to :ref:`--consul_tls_handshake_timeout <option_consul_tls_handshake_timeout_cmd>` command-line flag.

.. _option_pub_routines_cfg:

  * ``publisher_max_routines``: Equivalent to :ref:`--consul_publisher_max_routines <option_pub_routines_cmd>` command-line flag.

.. _yorc_config_file_terraform_section:

Terraform configuration
~~~~~~~~~~~~~~~~~~~~~~~

Below is an example of configuration file with Terraform configuration options.

.. code-block:: JSON

    {
      "resources_prefix": "yorc1-",
      "locations_file_path": "path-to-locations-yaml-or-json-config",
      "terraform": {
        "plugins_dir": "home/yorc/terraform_plugins_directory",
      }
    }

All available configuration options for Terraform are:

.. _option_plugins_dir_cfg:

  * ``plugins_dir``: Equivalent to :ref:`--terraform_plugins_dir <option_terraform_plugins_dir_cmd>` command-line flag.

.. _option_aws_plugin_version_constraint_cfg:

  * ``aws_plugin_version_constraint``: Equivalent to :ref:`--terraform_aws_plugin_version_constraint <option_terraform_aws_plugin_version_constraint_cmd>` command-line flag.

.. _option_consul_plugin_version_constraint_cfg:

  * ``consul_plugin_version_constraint``: Equivalent to :ref:`--terraform_consul_plugin_version_constraint <option_terraform_consul_plugin_version_constraint_cmd>` command-line flag.

.. _option_google_plugin_version_constraint_cfg:

  * ``google_plugin_version_constraint``: Equivalent to :ref:`--terraform_google_plugin_version_constraint <option_terraform_google_plugin_version_constraint_cmd>` command-line flag.

.. _option_openstack_plugin_version_constraint_cfg:

  * ``openstack_plugin_version_constraint``: Equivalent to :ref:`--terraform_openstack_plugin_version_constraint <option_terraform_openstack_plugin_version_constraint_cmd>` command-line flag.

.. _option_terraform_keep_generated_files_cfg:

  * ``keep_generated_files``: Equivalent to :ref:`--terraform_keep_generated_files <option_terraform_keep_generated_files_cmd>` command-line flag.


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
      "locations_file_path": "path-to-locations-yaml-or-json-config",
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

Tasks/Workers configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Below is an example of configuration file with Tasks configuration options.

.. code-block:: YAML

    resources_prefix: "yorc1-"
    tasks:
      dispatcher:
        long_polling_wait_time: "1m"
        lock_wait_time: "50ms"
        metrics_refresh_time: "5m"

.. _option_tasks_dispatcher_long_polling_wait_time_cfg:

  * ``long_polling_wait_time``: Equivalent to :ref:`--tasks_dispatcher_long_polling_wait_time <option_tasks_dispatcher_long_polling_wait_time_cmd>` command-line flag.

.. _option_tasks_dispatcher_lock_wait_time_cfg:

  * ``lock_wait_time``: Equivalent to :ref:`--tasks_dispatcher_lock_wait_time <option_tasks_dispatcher_lock_wait_time_cmd>` command-line flag.

.. _option_tasks_dispatcher_metrics_refresh_time_cfg:

  * ``metrics_refresh_time``: Equivalent to :ref:`--tasks_dispatcher_metrics_refresh_time <option_tasks_dispatcher_metrics_refresh_time_cmd>` command-line flag.


Environment variables
---------------------

.. _option_ansible_ssh_env:

  * ``YORC_ANSIBLE_USE_OPENSSH``: Equivalent to :ref:`--ansible_use_openssh <option_ansible_ssh_cmd>` command-line flag.

.. _option_ansible_debug_env:

  * ``YORC_ANSIBLE_DEBUG``: Equivalent to :ref:`--ansible_debug <option_ansible_debug_cmd>` command-line flag.

.. _option_ansible_connection_retries_env:

  * ``YORC_ANSIBLE_CONNECTION_RETRIES``: Equivalent to :ref:`--ansible_connection_retries <option_ansible_connection_retries_cmd>` command-line flag.

.. _option_ansible_cache_facts_env:

  * ``YORC_ANSIBLE_CACHE_FACTS``: Equivalent to :ref:`--ansible_cache_facts <option_ansible_cache_facts_cmd>` command-line flag.

.. _option_ansible_archive_artifacts_env:

  * ``YORC_ANSIBLE_JOB_MONITORING_TIME_INTERVAL``: Equivalent to :ref:`--ansible_job_monitoring_time_interval <option_ansible_job_monitoring_time_interval_cmd>` command-line flag.

.. _option_ansible_keep_generated_recipes_env:

  * ``YORC_ANSIBLE_KEEP_GENERATED_RECIPES``: Equivalent to :ref:`--ansible_keep_generated_recipes <option_ansible_keep_generated_recipes_cmd>` command-line flag.

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

.. _option_consul_tls_handshake_timeout_env:

  * ``YORC_CONSUL_TLS_HANDSHAKE_TIMEOUT``: Equivalent to :ref:`--consul_tls_handshake_timeout <option_consul_tls_handshake_timeout_cmd>` command-line flag.

.. _option_consul_store_txn_timeout_env:

  * ``YORC_CONSUL_STORE_TXN_TIMEOUT``: Allows to activate the feature that packs ConsulStore operations into transactions. If set to a valid Go duration, operations are packed into transactions up to 64 ops. This timeout represent the time to wait for new operations before sending an incomplete (less than 64 ops) transaction to Consul.

.. _option_pub_routines_env:

  * ``YORC_CONSUL_PUBLISHER_MAX_ROUTINES``: Equivalent to :ref:`--consul_publisher_max_routines <option_pub_routines_cmd>` command-line flag.

.. _option_shut_timeout_env:

  * ``YORC_SERVER_GRACEFUL_SHUTDOWN_TIMEOUT``: Equivalent to :ref:`--graceful_shutdown_timeout <option_shut_timeout_cmd>` command-line flag.

.. _option_wf_step_termination_timeout_env:

  * ``YORC_WF_STEP_GRACEFUL_TERMINATION_TIMEOUT``: Equivalent to :ref:`--wf_step_graceful_termination_timeout <option_wf_step_termination_timeout_cmd>` command-line flag.

.. _option_purged_deployments_eviction_timeout_env:

  * ``YORC_PURGED_DEPLOYMENTS_EVICTION_TIMEOUT``: Equivalent to :ref:`--purged_deployments_eviction_timeout <option_purged_deployments_eviction_timeout_cmd>` command-line flag.

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

  * ``YORC_SSL_VERIFY``: Equivalent to :ref:`--ssl_verify <option_ssl_verify_cmd>` command-line flag.

.. _option_ca_file_env:

  * ``YORC_CA_FILE``: Equivalent to :ref:`--ca_file <option_ca_file_cmd>` command-line flag.

.. _option_plugindir_env:

  * ``YORC_PLUGINS_DIRECTORY``: Equivalent to :ref:`--plugins_directory <option_pluginsdir_cmd>` command-line flag.

.. _option_resources_prefix_env:

  * ``YORC_RESOURCES_PREFIX``: Equivalent to :ref:`--resources_prefix <option_resources_prefix_cmd>` command-line flag.

.. _option_locations_env:

  * ``YORC_LOCATIONS_FILE_PATH``: Equivalent to :ref:`--locations_file_path <option_locations_cmd>` command-line flag.

.. _option_tasks_dispatcher_long_polling_wait_time_env:

  * ``YORC_TASKS_DISPATCHER_LONG_POLLING_WAIT_TIME``: Equivalent to :ref:`--tasks_dispatcher_long_polling_wait_time <option_tasks_dispatcher_long_polling_wait_time_cmd>` command-line flag.

.. _option_tasks_dispatcher_lock_wait_time_env:

  * ``YORC_TASKS_DISPATCHER_LOCK_WAIT_TIME``: Equivalent to :ref:`--tasks_dispatcher_lock_wait_time <option_tasks_dispatcher_lock_wait_time_cmd>` command-line flag.

.. _option_tasks_dispatcher_metrics_refresh_time_env:

  * ``YORC_TASKS_DISPATCHER_METRICS_REFRESH_TIME``: Equivalent to :ref:`--tasks_dispatcher_metrics_refresh_time <option_tasks_dispatcher_metrics_refresh_time_cmd>` command-line flag.

.. _option_workers_env:

  * ``YORC_WORKERS_NUMBER``: Equivalent to :ref:`--workers_number <option_workers_cmd>` command-line flag.

.. _option_workdir_env:

  * ``YORC_WORKING_DIRECTORY``: Equivalent to :ref:`--working_directory <option_workdir_cmd>` command-line flag.

.. _option_server_id_env:

  * ``YORC_SERVER_ID``: Equivalent to :ref:`--server_id <option_server_id_cmd>` command-line flag.

.. _option_disable_ssh_agent_env:

  * ``YORC_DISABLE_SSH_AGENT``: Equivalent to :ref:`--disable_ssh_agent <option_disable_ssh_agent_cmd>` command-line flag.

.. _option_upgrades_concurrency_env:

  * ``YORC_CONCURRENCY_LIMIT_FOR_UPGRADES``: Equivalent to :ref:`--concurrency_limit_for_upgrades <option_upgrades_concurrency_cmd>` command-line flag.

.. _option_ssh_connection_timeout_env:

  * ``YORC_SSH_CONNECTION_TIMEOUT``: Equivalent to :ref:`--ssh_connection_timeout <option_ssh_connection_timeout_cmd>` command-line flag.

.. _option_ssh_connection_retry_backoff_env:

  * ``YORC_SSH_CONNECTION_RETRY_BACKOFF``: Equivalent to :ref:`--ssh_connection_retry_backoff <option_ssh_connection_retry_backoff_cmd>` command-line flag.

.. _option_ssh_connection_max_retries_env:

  * ``YORC_SSH_CONNECTION_MAX_RETRIES``: Equivalent to :ref:`--ssh_connection_max_retries <option_ssh_connection_max_retries_cmd>` command-line flag.

.. _option_log_env:

  * ``YORC_LOG``: If set to ``1`` or ``DEBUG``, enables debug logging for Yorc.

.. _option_terraform_plugins_dir_env:

  * ``YORC_TERRAFORM_PLUGINS_DIR``: Equivalent to :ref:`--terraform_plugins_dir <option_terraform_plugins_dir_cmd>` command-line flag.

.. _option_terraform_aws_plugin_version_constraint:

  * ``YORC_TERRAFORM_AWS_PLUGIN_VERSION_CONSTRAINT``: Equivalent to :ref:`--terraform_aws_plugin_version_constraint <option_terraform_aws_plugin_version_constraint_cmd>` command-line flag.

.. _option_terraform_consul_plugin_version_constraint:

  * ``YORC_TERRAFORM_CONSUL_PLUGIN_VERSION_CONSTRAINT``: Equivalent to :ref:`--terraform_consul_plugin_version_constraint <option_terraform_consul_plugin_version_constraint_cmd>` command-line flag.

.. _option_terraform_google_plugin_version_constraint:

  * ``YORC_TERRAFORM_GOOGLE_PLUGIN_VERSION_CONSTRAINT``: Equivalent to :ref:`--terraform_google_plugin_version_constraint <option_terraform_google_plugin_version_constraint_cmd>` command-line flag.

.. _option_terraform_openstack_plugin_version_constraint:

  * ``YORC_TERRAFORM_OPENSTACK_PLUGIN_VERSION_CONSTRAINT``: Equivalent to :ref:`--terraform_openstack_plugin_version_constraint <option_terraform_openstack_plugin_version_constraint_cmd>` command-line flag.

.. _option_terraform_keep_generated_files_env:

  * ``YORC_TERRAFORM_KEEP_GENERATED_FILES``: Equivalent to :ref:`--terraform_keep_generated_files <option_terraform_keep_generated_files_cmd>` command-line flag.

.. _locations_configuration:

Locations configuration
-----------------------

A location allows Yorc to connect to an infrastructure. A location is identified uniquely by its ``name`` property.
Its ``type`` property Specifies the infrastructure related to this location. Yorc can handle multiple locations of the same infrastructure.

Its ``properties`` property contains a map with all required information for the infrastructure connection.

The :ref:`--locations_file_path option <option_locations_cmd>` allows user to define the specific locations configuration file path.
This configuration is taken in account for the first time the server starts and allows to populate locations for the Yorc cluster.
In near future, a REST API will let users add, remove or update existing locations configured in Yorc.

This file can be written either in JSON or YAML format.
Here is a JSON example of locations configuration:

.. code-block:: JSON

  {
  "locations": [
    { "name": "myOpenstackLocation1",
      "type": "openstack",
      "properties": {
        "auth_url": "http://openstack:5000/v2.0",
        "tenant_name": "Tname",
        "tenant_id": "use_tid_or_tname",
        "user_name": "{{with (secret \"/secret/yorc/mysecret\").Raw}}{{.Data.value}}{{end}}",
        "password": "{{secret \"/secret/yorc/mysecret\" \"data=value\" | print}}",
        "region": "RegionOne",
        "private_network_name": "private-test",
        "public_network_name": "not_supported",
        "os_default_security_groups": ["default", "lax"]
      }
    },
    { "name": "myGoogleLocation1",
      "type": "google",
      "properties": {
        "application_credentials": "creds.json",
        "project": "my-project-ref"
      }
    },
    ....

Builtin locations types configuration
-------------------------------------

Here we have principal infrastructure configurations retrieved as location properties for a specified type.

.. _option_infra_os:

OpenStack
~~~~~~~~~

OpenStack location type is ``openstack`` in lower case.

..
   MAG - According to:
   https://github.com/sphinx-doc/sphinx/issues/3043
   http://www.sphinx-doc.org/en/stable/markup/misc.html#tables
.. tabularcolumns:: |p{0.35\textwidth}|p{0.30\textwidth}|p{0.05\textwidth}|p{0.15\textwidth}|p{0.10\textwidth}|

+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
|            Property Name          |                                                     Description                                                     | Data Type |                      Required                      |    Default    |
|                                   |                                                                                                                     |           |                                                    |               |
+===================================+=====================================================================================================================+===========+====================================================+===============+
| ``auth_url``                      | Specify the authentication url for OpenStack (should be the Keystone endpoint ie: http://your-openstack:5000/v2.0). | string    | yes                                                |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``tenant_id``                     | Specify the OpenStack tenant id to use.                                                                             | string    | Either this or ``tenant_name`` should be provided. |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``tenant_name``                   | Specify the OpenStack tenant name to use.                                                                           | string    | Either this or ``tenant_id`` should be provided.   |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``user_domain_name``              | Specify the domain name where the user is located (Identity v3 only).                                               | string    | yes (if use Identity v3)                           |               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``project_id``                    | Specify the ID of the project to login with (Identity v3 only).                                                     | string    | Either this or ``project_name`` should be provided.|               |
+-----------------------------------+---------------------------------------------------------------------------------------------------------------------+-----------+----------------------------------------------------+---------------+
| ``project_name``                  | Specify the name of the project to login with (Identity v3 only).                                                   | string    | Either this or ``project_id`` should be provided.  |               |
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

Kubernetes location type is ``kubernetes`` in lower case.

..
   MAG - According to:
   https://github.com/sphinx-doc/sphinx/issues/3043
   http://www.sphinx-doc.org/en/stable/markup/misc.html#tables
.. tabularcolumns:: |l|L|L|L|L|

+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
|           Property Name          |                                   Description                                   | Data Type | Required | Default |
|                                  |                                                                                 |           |          |         |
+==================================+=================================================================================+===========+==========+=========+
| ``kubeconfig``                   | Path or content of Kubernetes cluster configuration file*                       | string    | no       |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``application_credentials``      | Path or content of file containing credentials**                                | string    | no       |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``master_url``                   | URL of the HTTP API of Kubernetes is exposed. Format: ``https://<host>:<port>`` | string    | no       |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``ca_file``                      | Path to a trusted root certificates for server                                  | string    | no       |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``cert_file``                    | Path to the TLS client certificate used for authentication                      | string    | no       |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``key_file``                     | Path to the TLS client key used for authentication                              | string    | no       |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``insecure``                     | Server should be accessed without verifying the TLS certificate (testing only)  | boolean   | no       |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+
| ``job_monitoring_time_interval`` | Default duration for job monitoring time interval                               | string    | no       | 5s      |
+----------------------------------+---------------------------------------------------------------------------------+-----------+----------+---------+

* ``kubeconfig`` is the path (accessible to Yorc server) or the content of a Kubernetes
  cluster configuration file.
  When ``kubeconfig`` is defined, other infrastructure configuration properties (``master_url``,
  keys or certificates) don't have to be defined here.

  If neither ``kubeconfig`` nor ``master_url`` is specified, the Orchestrator will
  consider it is running within a Kubernetes Cluster and will attempt to authenticate
  inside this cluster.

* ``application_credentials`` is the path (accessible to Yorc server) or the content
  of a file containing Google service account private keys in JSON format.
  This file can be downloaded from the Google Cloud Console at  `Google Cloud service account file <https://console.cloud.google.com/apis/credentials/serviceaccountkey>`_.
  It is needed to authenticate against Google Cloud when the ``kubeconfig`` property
  above refers to a Kubernetes Cluster created on Google Kubernetes Engine, and the orchestrator is running on a host
  where `gcloud <https://cloud.google.com/sdk/gcloud/>`_ is not installed.

.. _option_infra_google:

Google Cloud Platform
~~~~~~~~~~~~~~~~~~~~~

Google Cloud Platform location type is ``google`` in lower case.

+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+
|  Property Name              |              Description                     | Data Type | Required | Default                                |
|                             |                                              |           |          |                                        |
+=============================+==============================================+===========+==========+========================================+
| ``project``                 | ID of the project to apply any resources to  | string    | yes      |                                        |
+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+
| ``application_credentials`` | Path of file containing credentials*         | string    | no       | Google Application Default Credentials |
+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+
| ``credentials``             | Content of file containing credentials       | string    | no       | Google Application Default Credentials |
+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+
| ``region``                  | The region to operate under                  | string    | no       |                                        |
+-----------------------------+----------------------------------------------+-----------+----------+----------------------------------------+

``application_credentials`` is the path (accessible to Yorc server) of a file containing service account private keys in JSON format.
This file can be downloaded from the Google Cloud Console at  `Google Cloud service account file <https://console.cloud.google.com/apis/credentials/serviceaccountkey>`_.

If no file path is specified in ``application_credentials`` and no file content is specified in ``credentials``, the orchestrator will fall back to using the `Google Application Default Credentials <https://cloud.google.com/docs/authentication/production>`_ if any.

.. _option_infra_aws:

AWS
~~~

AWS location type is ``aws`` in lower case.

+----------------+----------------------------------------+-----------+----------+---------+
|  Property Name |              Description               | Data Type | Required | Default |
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
~~~~~

Slurm location type is ``slurm`` in lower case.

+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
|          Property Name           |                                   Description                                   | Data Type |                     Required                      | Default |
|                                  |                                                                                 |           |                                                   |         |
+==================================+=================================================================================+===========+===================================================+=========+
| ``user_name``                    | SSH Username to be used to connect to the Slurm Client's node                   | string    | yes (see below for alternatives)                  |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``password``                     | SSH Password to be used to connect to the Slurm Client's node                   | string    | Either this or ``private_key`` should be provided |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``private_key``                  | SSH Private key to be used to connect to the Slurm Client's node                | string    | Either this or ``password`` should be provided    |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``url``                          | IP address of the Slurm Client's node                                           | string    | yes                                               |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``port``                         | SSH Port to be used to connect to the Slurm Client's node                       | string    | yes                                               |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``default_job_name``             | Default name for the job allocation.                                            | string    | no                                                |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``job_monitoring_time_interval`` | Default duration for job monitoring time interval                               | string    | no                                                | 5s      |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``enforce_accounting``           | If true, account properties are mandatory for jobs and computes                 | boolean   | no                                                | false   |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``keep_job_remote_artifacts``    | If true, job artifacts are not deleted at the end of the job.                   | boolean   | no                                                | false   |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``ssh_connection_timeout``       | Allow to supersede                                                              | Duration  | no                                                | false   |
|                                  | :ref:`--ssh_connection_timeout <option_ssh_connection_timeout_cmd>`             |           |                                                   |         |
|                                  | global server option for this specific location.                                |           |                                                   |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``ssh_connection_retry_backoff`` | Allow to supersede                                                              | Duration  | no                                                | false   |
|                                  | :ref:`--ssh_connection_retry_backoff <option_ssh_connection_retry_backoff_cmd>` |           |                                                   |         |
|                                  | global server option for this specific location.                                |           |                                                   |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+
| ``ssh_connection_max_retries``   | Allow to supersede                                                              | uint64    | no                                                | false   |
|                                  | :ref:`--ssh_connection_max_retries <option_ssh_connection_max_retries_cmd>`     |           |                                                   |         |
|                                  | global server option for this specific location.                                |           |                                                   |         |
+----------------------------------+---------------------------------------------------------------------------------+-----------+---------------------------------------------------+---------+

An alternative way to specify user credentials for SSH connection to the Slurm Client's node (user_name, password or private_key), is to provide them as application properties.
In this case, Yorc gives priority to the application provided properties.
Moreover, if all the applications provide their own user credentials, the configuration properties user_name, password and private_key, can be omitted.
See `Working with jobs <https://yorc-a4c-plugin.readthedocs.io/en/latest/jobs.html>`_ for more information.

.. _option_storage_config:

Storage configuration
---------------------

Different artifacts (topologies, logs, events, tasks...) are stored by Yorc during an application deployment.

Previously, everything was stored in Consul KV.
Starting with version 4.0.0, we choosed to refactor the way Yorc stores data mainly for performance reasons, and also to make it more flexible.
Yorc can now store the different kind of artifacts in different ``stores`` configured in a new section of the configuration file called ``storage``.

If defined, the ``storage`` entry may specify the following properties:
 * the ``stores`` property allows to customize storage in a different way than the default one.
 * the ``default_properties`` allows to change properties settings for the default fileCache store.
 * The ``reset`` property allows to redefine the stores or to change properties for default stores when Yorc re-starts. If no set to true, the existing storage is used.

+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
|     Property Name                |                          Description                             | Data Type |   Required       | Default         |
|                                  |                                                                  |           |                  |                 |
+==================================+==================================================================+===========+==================+=================+
| ``reset``                        | See :ref:`Storage reset note <storage_reset_note>`               | boolean   | no               |   False         |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
| ``default_properties``           | Properties for default fileCache store.                          |           |                  |                 |
|                                  | See :ref:`File cache properties <storage_file_cache_props>`      | map       | no               |                 |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
| ``stores``                       | Stores configuration                                             | array     | no               | See Store types |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+

So now, users can configure different store types for storing the different kind of artifacts, and using different stores implementations.

Currently Yorc supports 3 store ``types``:
  * ``Deployment``
  * ``Log``
  * ``Event``

Yorc supports 5 store ``implementations``:
  * ``consul``
  * ``file``
  * ``cipherFile``
  * ``fileCache``
  * ``cipherFileCache``
  * ``elastic`` (experimental)

By default, ``Log`` and ``Event`` store types use ``consul`` implementation, and ``Deployment`` store uses ``fileCache``.

If these default settings correspond to your needs, the Yorc configuration file does not need to have a ``storage`` entry.

If you want to change properties for the default ``fileCache`` store for ``Deployment``, you have to set the new values in the ``default_properties`` map.
The ``cache_num_counters`` and ``cache_max_cost`` properties can be used to determine the cache size in function of the expected number of items.
The default values are defined for about 100 deployments if we approximate a cache size of 100 K and 100 items for one single deployment.
See `Default cache size for file storage is too large  <https://github.com/ystia/yorc/issues/612>`_.

Pay attention that the cache size must be defined in function of the Yorc host memory resources and a too large cache size can affect performances.


Here is a JSON example of updating default properties for cache used in ``fileCache`` store for ``Deployment``:

.. code-block:: JSON

  {
  "storage": {
    "reset": true,
    "default_properties":
    {
      "cache_num_counters": 1e7,
      "cache_max_cost": 1e9
    }
   }
  }

The same sample in YAML

.. code-block:: YAML

    storage:
      reset: true
      default_properties:
        cache_num_counters: 1e7
        cache_max_cost: 1e9


A store configuration is defined with:

+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
|     Property Name                |                          Description                             | Data Type |   Required       | Default         |
|                                  |                                                                  |           |                  |                 |
+==================================+==================================================================+===========+==================+=================+
| ``name``                         | unique store ID                                                  | string    | no               |  generated      |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
| ``migrate_data_from_consul``     | Log and Event data migration from consul. See note below.        | bool      | no               | false           |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
| ``implementation``               | Store implementation. See Store implementations below.           | string    | yes              |                 |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
| ``types``                        | Store types handled by this instance. See Store types below.     | array     | yes              |                 |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+
| ``properties``                   | Specific store implementation properties.                        | map       | no               |                 |
+----------------------------------+------------------------------------------------------------------+-----------+------------------+-----------------+

``migrate_data_from_consul`` allows to migrate data from ``consul`` to another store implementation.
This is useful when a new store is configured (different from consul...) for logs or events.


Store types
~~~~~~~~~~~

Currently 3 different store types are supported by Yorc:

Deployment
^^^^^^^^^^

This store type contains data representing the Tosca topologies types (data, nodes, policies, artifacts, relationships, capabilities) and templates (nodes, polocies, repositories, imports, workflows).

Data in this store is written once when a topology is parsed, then read many times during application lifecycle. ``fileCache`` is the default implementation for this store type.

Log
^^^

Store that contains the applicative logs, also present in Alien4Cloud logs. ``consul`` is the default implementation for this store type.

If you face some Consul memory usage issues, you can choose ``file`` or ``cipherFile`` as logs may contains private information.

Cache is not useful for this kind of data as we use blocking queries and modification index to retrieve it.

Event
^^^^^

Store that contains the applicative events, also present in Alien4Cloud events. ``consul`` is the default implementation for this store type.
Same remarks as for Log as it's same kind of data and usage.

Store implementations
~~~~~~~~~~~~~~~~~~~~~

Currently Yorc provide 5 implementations (in fact 2 real ones with combinations around file) described below but you're welcome to contribute and bring your own implementation, you just need to implement the Store interface
See `Storage interface  <https://github.com/ystia/yorc/blob/develop/storage/store/store.go>`_.

consul
^^^^^^

This is the Consul KV store used by Yorc for main internal storage stuff. For example, the configuration of the stores is kept in the Consul KV.
As Consul is already configurable here: :ref:`Consul configuration<yorc_config_file_consul_section>`, no other configuration is provided in this section.

file
^^^^

This is a file store without cache system. It can be used for logs and events as this data is retrieved via blocking queries and modification index which can be used with a cache system.

Here are specific properties for this implementation:

.. _storage_file_props:

+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+
|     Property Name                   |           Description                              | Data Type |   Required       | Default         |
+=====================================+====================================================+===========+==================+=================+
| ``root_dir``                        | Root directory used for file storage               | string    | no               |   work/store    |
+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``blocking_query_default_timeout``  | default timeout for blocking queries               | string    | no               |   5m (5 minutes)|
+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``concurrency_limit``               | Limit for concurrent operations                    | integer   | no               |   1000          |
+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+

cipherFile
^^^^^^^^^^

This is a file store with file data encryption (AES-256 bits key) which requires a 32-bits length passphrase.

Here are specific properties for this implementation in addition to ``file`` properties:

+----------------------------------+----------------------------------------------------+-----------+------------------+-----------------+
|     Property Name                |           Description                              | Data Type |   Required       | Default         |
+==================================+====================================================+===========+==================+=================+
| ``passphrase``                   | Passphrase used to generate the encryption key     | string    | yes              |                 |
|                                  | Required to be 32-bits length                      |           |                  |                 |
+----------------------------------+----------------------------------------------------+-----------+------------------+-----------------+


``Passphrase`` can be set with ``Secret function`` and retrieved from Vault as explained in the Vault integration chapter.


Here is a JSON example of stores configuration with a cipherFile store implementation for logs.

.. code-block:: JSON

  {
  "storage": {
    "reset": false,
    "stores": [
      {
        "name": "myCipherFileStore",
        "implementation": "cipherFile",
        "migrate_data_from_consul": true,
        "types":  ["Log"],
        "properties": {
          "root_dir": "/mypath/to/store",
          "passphrase": "myverystrongpasswordo32bitlength"
        }
      }
   ]}
  }

The same sample in YAML

.. code-block:: YAML

    storage:
      reset: false
      stores:
      - name: myCipherFileStore
        implementation: cipherFile
        migrate_data_from_consul: true
        types:
        - Log
        properties:
          root_dir: "/mypath/to/store"
          passphrase: "myverystrongpasswordo32bitlength"

fileCache
^^^^^^^^^

This is a file store with a cache system.

Here are specific properties for this implementation:

.. _storage_file_cache_props:

+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+
|     Property Name                   |           Description                              | Data Type |   Required       | Default         |
+=====================================+====================================================+===========+==================+=================+
| ``cache_num_counters``              | number of keys to track frequency of               | int64     | no               |   1e5 (100 000) |
+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``cache_max_cost``                  | maximum cost of cache                              | int64     | no               |   1e7 (10 M)    |
+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``cache_buffer_items``              | number of keys per Get buffer                      | int64     | no               |   64            |
+-------------------------------------+----------------------------------------------------+-----------+------------------+-----------------+

For more information on cache properties, you can refer to `Ristretto README  <https://github.com/dgraph-io/ristretto>`_

cipherFileCache
^^^^^^^^^^^^^^^

This is a file store with a cache system and file data encryption (AES-256 bits key) which requires a 32-bits length passphrase.


.. _storage_reset_note:

Stores configuration is saved once when Yorc server starts. If you want to re-initialize storage, you have to set the ``reset`` property to True and restart Yorc.

.. warning::
    Pay attention that if any data is still existing after reset, Yorc will ignore it.

If no storage configuration is set, default stores implementations are used as defined previously to handle all store types (``Deployment``, ``Log`` and ``Event``).

If any storage configuration is set with partial stores types, the missing store types will be added with default implementations.

elastic
^^^^^^^

This store ables you to store ``Log`` s and ``Event`` s in elasticsearch.

.. warning::
    This storage is only suitable to store logs and events.

 Per Yorc cluster : 1 index for logs, 1 index for events.

+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
|     Property Name           |           Description                              | Data Type |   Required       | Default         |
+=============================+====================================================+===========+==================+=================+
| ``es_urls``                 | the ES cluster urls                                | []string  | yes              |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``ca_cert_path``            | path to the PEM encoded CA's certificate file when | string    | no               |                 |
|                             | TLS is activated for ES                            |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``cert_path``               | path to a PEM encoded certificate file when TLS    | string    | no               |                 |
|                             | is activated for ES                                |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``key_path``                | path to a PEM encoded private key file when TLS    | string    | no               |                 |
|                             | is activated for ES                                |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``index_prefix``            | indexes used by yorc can be prefixed               | string    | no               |   yorc_         |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``es_query_period``         | when querying logs and event, we wait this timeout | duration  | no               |   4s            |
|                             | before each request when it returns nothing (until |           |                  |                 |
|                             | something is returned or the waitTimeout is        |           |                  |                 |
|                             | reached)                                           |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``es_refresh_wait_timeout`` | used to wait for more than refresh_interval (1s)   | duration  | no               |   2s            |
|                             | (until something is returned or the waitTimeout is |           |                  |                 |
|                             | is reached)                                        |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``es_force_refresh``        | when querying ES, force refresh index before when  | bool      | no               |   false         |
|                             | waiting for refresh.                               |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``max_bulk_size``           | the maximum size (in kB) of bulk request sent when | int64     | no               |   4000          |
|                             | while migrating data                               |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``max_bulk_count``          | maximum size (in term of number of documents) when | int64     | no               |   1000          |
|                             | of bulk request sent while migrating data          |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``cluster_id``              | used to distinguish logs & events in the indexes   | string    | no               |                 |
|                             | if different yorc cluster are writing in the same  |           |                  |                 |
|                             | elastic cluster.                                   |           |                  |                 |
|                             | If not set, the consul.datacenter will be used.    |           |                  |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``trace_requests``          | to print ES requests (for debug only)              | bool      | no               |   false         |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``trace_events``            | to trace events & logs when sent (for debug only)  | bool      | no               |   false         |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``initial_shards``          | number of shards used to initialize indices        | int64     | no               |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+
| ``initial_replicas``        | number of replicas used to initialize indices      | int64     | no               |                 |
+-----------------------------+----------------------------------------------------+-----------+------------------+-----------------+


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

.. _option_client_tls_cmd:

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

.. _option_client_tls_cfg:

  * ``ssl_enabled``: Equivalent to :ref:`--ssl_enabled <option_client_tls_cmd>` command-line flag.

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

.. _option_client_tls_env:

  * ``YORC_SSL_ENABLED``: Equivalent to :ref:`--ssl_enabled <option_client_tls_cmd>` command-line flag.

.. _option_client_yorc_api_env:

  * ``YORC_API``: Equivalent to :ref:`--yorc_api <option_client_yorc_api_cmd>` command-line flag.
