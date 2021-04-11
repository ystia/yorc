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

.. _yorc_bootstrap_section:

Bootstrap a full stack installation
===================================

The command ``yorc bootstrap`` can be used to bootstrap the full stack, from Alien4Cloud
to Yorc and its dependencies, over different types of infrastructures, on a single node
or distributed on several nodes.

Prerequisites
-------------

Hosts
~~~~~

The local host from where the command ``yorc bootstrap`` will be run, as well as
remote hosts where the full stack will be deployed, should be Linux x86_64 systems
operating with at least 2 CPUs and 4 Go of RAM.
The bootstrap was validated on:

  * CentOS 7,
  * Red Hat Enterprise Linux 7.5,
  * Ubuntu 19.04 (which is installing python3 by default, see
    :ref:`bootstrap configuration file <yorc_google_example_ubuntu_section>`
    example below for specific Ansible configuration settings needed for remote
    hosts using python3).


.. warning:: For Centos and RHEL an access to an EPEL repository is required in order to install pip.

Packages
~~~~~~~~

The bootstrap operation will install a basic Yorc setup on the local host.

It requires the following packages to be installed on the local host:

  * python and python-pip (or python3/python3-pip)
  * zip/unzip
  * openssh-client
  * openssl (to generate certificates when they are not provided)
  * wget

This basic installation on the local host will attempt to install without sudo privileges
python ansible module |ansible_version| if needed as well as paramiko and python packages
MarkupSafe, jinja2, PyYAML, six, cryptography, setuptools.
So if this ansible module or python packages are not yet installed on the local host,
you could either add them yourself, with sudo privileges, running for example:

.. parsed-literal::

    sudo pip install ansible==\ |ansible_version|

Or you could create a python virtual environment, and let the Yorc bootstrap command
install the ansible module within this virtual environment (operation which doesn't require sudo privileges).

You can run these commands to create a virtual environment, here a virtual
environment called ``yorcenv``:

.. parsed-literal::

    sudo pip install virtualenv
    virtualenv yorcenv
    source yorcenv/bin/activate

You are now ready to download Yorc binary, running:

.. parsed-literal::

    wget \https://github.com/ystia/yorc/releases/download/v\ |yorc_version|\ /yorc-\ |yorc_version|\ .tgz
    tar xzf yorc-\ |yorc_version|\ .tgz
    ./yorc bootstrap --help

Define firewall rules on your Cloud Provider infrastructure
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

To access Alien4Cloud UI from the local host, you may need to define firewall
rules before attempting to bootstrap the full stack on a cloud provider infrastructure.

For example on Google Cloud, you could define this firewall rule for the port 8088
used by the UI, and associate it to a tag (here ``a4c``) :

.. code-block:: bash

    $ gcloud compute firewall-rules create a4c-rule \
      --allow tcp:8088 --target-tags a4c

You could then specify this tag ``a4c`` in the compute instance to create by the
bootstrap deployment, as it is done in :ref:`Google Configuration file example <yorc_google_example_section>`.
This way the created compute instance where Alien4Cloud will be deployed will
have its port 8088 open.

Bootstrap process overview
--------------------------

The command ``yorc bootstrap`` will configure on the local host a basic setup with
Yorc and a Consul data store.

This basic setup will then be used to bootstrap the full stack Alien4Cloud/Yorc
and its dependencies over a selected location.

You can deploy the full stack either on a single node (by default), or distributed
on several nodes as described in :ref:`Run Yorc in HA mode <yorc_ha_section>`, using ``yorc bootstrap``
command line option ``--deployment_type HA`` described below.

When flag ``--insecure`` is not specified, a secured installation will performed:

  * TLS with mutual authentication between components will be configured,
  * a Vault will be installed and used to store location credentials.

Configuration values can be provided by the user:

 * in interactive mode,
 * through a configuration file,
 * using ``yorc bootstrap`` command line options
 * using environment variables.

You can combine these modes, ``yorc bootstrap`` will check in any case if
required configuration values are missing, and will ask for missing values.

These configuration values will allow you to specify:

  * optional Alien4Cloud configuration values
  * Yorc configuration values, all optional, except from:
      * the path to a ssh private key that will be used by the local orchestrator to connect to the bootstrapped setup
      * the Certificate authority private key passphrase to use in the default secure mode
        (while the other properties, Certificate Authority private key and PEM-encoded Certificate Authority, are optional. If not provided, they will be generated, and the generated Certificate Authority at ``work/bootstrapResources/ca.pem`` can then be imported in your Web browser as a trusted Certificate Authority)
  * Location configuration with required configuration values depending on
    the infrastructure type, as described at :ref:`Locations Configuration <locations_configuration>`
  * Configuration of compute Nodes to create on demand,
  * User used to connect to these compute nodes,
  * Configuration of the connection to public network created on demand.

Details of these on-demand resources configuration values are provided in the Alien4Cloud
Yorc plugin Documentation at https://yorc-a4c-plugin.readthedocs.io/en/latest/location.html.
For example, in the :ref:`Google Configuration file example <yorc_google_example_section>`, you can see on-demand ``compute``  and ``address`` configuration values.

Once configuration settings are provided, ``yorc bootstrap`` will proceed to the
full stack deployment, showing deployment steps progress (by default, but you can see
deployment logs instead trough the option ``--follow logs`` described below).

Once the deployment is finished, the orchestrator on the local host is still running,
so you can perform commands like ``./yorc deployments list``, ``./yorc deployments logs -b``, etc...
Or perform any deployment troubleshooting if needed.

To undeploy a bootstrapped setup, you can also use the CLI, running ``./yorc deployments undeploy <deployment id>``.

To clean the local host setup, run:

.. parsed-literal::

    ./yorc bootstrap cleanup

This will only clean the local host environment, it won't undeploy the bootstrapped
setup installed on remote hosts.
It stops the local yorc and consul servers, cleans files in working directory except from downloaded bundles, on purpose as some of them take time to be downloaded.

Bootstrapping the setup in interactive mode
-------------------------------------------

You can bootstrap the setup in interactive mode running:

.. parsed-literal::

    ./yorc bootstrap [--review]

You will have then to select the infrastructure type (Google Cloud, AWS,
OpenStack, Hosts Pool) and provide a name to the location on which you want to deploy the full stack, then you will
be asked to provide configuration values depending on the infrastructure type.

The command line option ``--review`` allows to review and update all configuration
values before proceeding to the deployment, opening the editor specified in the
environment variable ``EDITOR`` if defined or using vi or vim if available.

Bootstrapping the setup using command line options
--------------------------------------------------

The following ``yorc bootstrap`` option are available:

  * ``--alien4cloud_download_url`` Alien4Cloud download URL (defaults to the Alien4Cloud version compatible with this Yorc, under https://www.portaildulibre.fr/nexus/repository/opensource-releases/alien4cloud/alien4cloud-premium-dist/)
  * ``--alien4cloud_password`` Alien4Cloud password (default, admin)
  * ``--alien4cloud_port`` Alien4Cloud port (default 8088)
  * ``--alien4cloud_user`` Alien4Cloud user (default, admin)
  * ``--ansible_extra_package_repository_url`` URL of package indexes where to find the ansible package, instead of the default Python Package repository
  * ``--ansible_use_openssh`` Prefer OpenSSH over Paramiko, python implementation of SSH
  * ``--ansible_version`` Ansible version (default \ |ansible_version|\ )
  * ``--config_only`` Makes the bootstrapping abort right after exporting the inputs
  * ``--consul_download_url`` Consul download URL (default, Consul version compatible with this Yorc, under https://releases.hashicorp.com/consul/)
  * ``--consul_encrypt_key`` 16-bytes, Base64 encoded value of an encryption key used to encrypt Consul network traffic
  * ``--consul_port`` Consul port (default 8543)
  * ``--credentials_user`` User Yorc uses to connect to Compute Nodes
  * ``--deployment_name`` Name of the deployment. If not specified deployment name is based on time.
  * ``--deployment_type`` Define deployment type: single_node or HA (default, single_node)
  * ``--follow`` Follow bootstrap deployment steps, logs, or none (default, steps)
  * ``--infrastructure`` Define the type of infrastructure where to deploy Yorc: google, openstack, aws, hostspool
  * ``--insecure`` Insecure mode - no TLS configuration, no Vault to store secrets
  * ``--jdk_download_url`` Java Development Kit download URL (default, JDK downloaded from https://edelivery.oracle.com/otn-pub/java/jdk/)
  * ``--jdk_version`` Java Development Kit version (default 1.8.0-131-b11)
  * ``--location`` Name identifying the location where to deploy Yorc
  * ``--resources_zip`` Path to bootstrap resources zip file (default, zip bundled within Yorc)
  * ``--review`` Review and update input values before starting the bootstrap
  * ``--terraform_download_url`` Terraform download URL (default, Terraform version compatible with this Yorc, under https://releases.hashicorp.com/terraform/)
  * ``--terraform_plugins_download_urls`` Terraform plugins download URLs (default, Terraform plugins compatible with this Yorc, under https://releases.hashicorp.com/terraform-provider-xxx/)
  * ``--values`` Path to file containing input values
  * ``--vault_download_url`` Hashicorp Vault download URL (default "https://releases.hashicorp.com/vault/1.0.3/vault_1.0.3_linux_amd64.zip")
  * ``--vault_port`` Vault port (default 8200)
  * ``--working_directory`` Working directory where to place deployment files (default, work)
  * ``--yorc_ca_key_file`` Path to Certificate Authority private key, accessible locally
  * ``--yorc_ca_passphrase`` Bootstrapped Yorc Home directory (default, /var/yorc)
  * ``--yorc_ca_pem_file`` Path to PEM-encoded Certificate Authority, accessible locally
  * ``--yorc_data_dir`` Bootstrapped Yorc Home directory (default, /var/yorc)
  * ``--yorc_download_url`` Yorc download URL (default, current Yorc release under https://github.com/ystia/yorc/releases/)
  * ``--yorc_plugin_download_url`` Yorc plugin download URL
  * ``--yorc_port`` Yorc HTTP REST API port (default 8800)
  * ``--yorc_private_key_file`` Path to ssh private key accessible locally
  * ``--yorc_workers_number`` Number of Yorc workers handling bootstrap deployment tasks (default 30)

The option ``--resources_zip`` is an advanced usage option allowing you to change
the bootstrap deployment description. You need to clone first the Yorc source code repository at
https://github.com/ystia/yorc, go into to directory ``commands``, change deployment
description files under ``bootstrap/resources/topology``, then zip the content of ``bootstrap/resources/``
so that this zip will be used to perform the bootstrap deployment.

Bootstrapping the setup using environment variables
---------------------------------------------------

Similarly to the configuration of ``yorc server`` through environment variables
described at :ref:`Yorc Server Configuration <yorc_config_section>`, the bootstrap configuration can be provided
through environment variables following the same naming rules, for example:

  * ``YORC_ALIEN4CLOUD_PORT`` allows to define the Alien4Cloud port

Once these environment variables are defined, you can bootstrap the setup running :
.. parsed-literal::

    ./yorc bootstrap [--review]

Bootstrapping the setup using a configuration file
--------------------------------------------------

You can bootstrap the setup using a configuration file running:

.. parsed-literal::

    ./yorc bootstrap --values <path to configuration file> [--review]

Similarly to the configuration of ``yorc server`` through a configuration file,
described at :ref:`Yorc Server Configuration <yorc_config_section>`, the bootstrap configuration can be provided
in a configuration file following the same naming rules for configuration variables,
for example :

.. code-block:: YAML

  alien4cloud:
    user: admin
    port: 8088
  locations:
    - name: myLocation
      type: openstack
      properties:
        auth_url: http://10.197.135.201:5000/v2.0


The bootstrap configuration file can be also be used to define Ansible Inventory
configuration parameters.
This is needed for example if you want to use on target hosts a python version
different from the one automatically selected by Ansible.

In this case, you can add in the bootstrap configuration file, a section allowing
to configure an Ansible behavioral inventory parameter that will allow to specify
which python interpreter could be used by Ansible on target hosts, as described in
:ref:`Ansible Inventory Configuration section <option_ansible_inventory_cfg>`.

This would give for example in the bootstrap configuration file:

.. code-block:: YAML

  ansible:
    inventory:
      "target_hosts:vars":
      - ansible_python_interpreter=/usr/bin/python3

See later below a :ref:`full example of bootstrap configuration file <yorc_google_example_ubuntu_section>` defining such a parameter.

Sections below provide examples of configuration files define a location for each infrastructure type.

.. _yorc_google_example_section:

Example of a Google Cloud deployment configuration file
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: YAML

  yorc:
    # Path to private key file on local host
    # used to connect to hosts on the bootstrapped setup
    private_key_file: /home/myuser/.ssh/yorc.pem
    # Path to Certificate Authority private key, accessible locally
    # If no key ile provided, one will be generated
    ca_key_file: /home/myuser//ca-key.pem
    # Certificate authority private key passphrase
    ca_passphrase: changeme
    # Path to PEM-encoded Certificate Authority, accessible locally
    # If not provided, a Certifcate Authority will be generated
    ca_pem_file: /home/myuser/ca.pem
  locations:
    - name: firstGoogleLocation
      type: google
      properties:
        # Path on local host to file containing Google service account private keys
        application_credentials: /home/myuser/gcp/myproject-a90a&bf599ef.json
        project: myproject
  address:
    region: europe-west1
  compute:
    image_project: centos-cloud
    image_family: centos-7
    machine_type: n1-standard-2
    zone: europe-west1-b
    # User and public key to define on created compute instance
    metadata: "ssh-keys=user1:ssh-ed25519 AAAABCd/gV/C+b3h3r5K011evEELMD72S4..."
    tags: a4c
  credentials:
    # User on compute instance created on demand
    user: user1


.. _yorc_google_example_ubuntu_section:

Example of a Google Cloud deployment configuration enforcing the use of python3
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In this example, a specific Ansible behavioral inventory parameter
``ansible_python_interpreter`` is defined so that Ansible will use the specified
python interpreter on the target hosts.

.. code-block:: YAML

  yorc:
    # Path to private key file on local host
    # used to connect to hosts on the bootstrapped setup
    private_key_file: /home/myuser/.ssh/yorc.pem
    # Path to Certificate Authority private key, accessible locally
    # If no key ile provided, one will be generated
    ca_key_file: /home/myuser//ca-key.pem
    # Certificate authority private key passphrase
    ca_passphrase: changeme
    # Path to PEM-encoded Certificate Authority, accessible locally
    # If not provided, a Certifcate Authority will be generated
    ca_pem_file: /home/myuser/ca.pem
  locations:
    - name: firstGoogleLocation
      type: google
      properties:
        # Path on local host to file containing Google service account private keys
        application_credentials: /home/myuser/gcp/myproject-a90a&bf599ef.json
        project: myproject
  ansible:
    inventory:
      # Enforce the use of /usr/bin/python3 by Ansible on target hosts
      "target_hosts:vars":
      - ansible_python_interpreter=/usr/bin/python3
  address:
    region: europe-west1
  compute:
    image_project: ubuntu-os-cloud
    image_family: ubuntu-1904
    machine_type: n1-standard-2
    zone: europe-west1-b
    # User and public key to define on created compute instance
    metadata: "ssh-keys=user1:ssh-ed25519 AAAABCd/gV/C+b3h3r5K011evEELMD72S4..."
    tags: a4c
  credentials:
    # User on compute instance created on demand
    user: user1

Example of an AWS deployment configuration file
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: YAML

  yorc:
    # Path to private key file on local host
    # used to connect to hosts on the bootstrapped setup
    private_key_file: /home/myuser/.ssh/yorc.pem
    # Path to Certificate Authority private key, accessible locally
    # If no key ile provided, one will be generated
    ca_key_file: /home/myuser//ca-key.pem
    # Certificate authority private key passphrase
    ca_passphrase: changeme
    # Path to PEM-encoded Certificate Authority, accessible locally
    # If not provided, a Certifcate Authority will be generated
    ca_pem_file: /home/myuser/ca.pem
  locations:
    - name: firstAWSLocation
      type: aws
      properties:
        region: us-east-2
        access_key: ABCDEFABCDEFABCD12DA
        secret_key: aabcdxYxABC/a1bcdef
  address:
    ip_version: 4
  compute:
    image_id: ami-18f8df7d
    instance_type: t2.large
    key_name: key-yorc
    security_groups: janus-securityGroup
    delete_volume_on_termination: true
  credentials:
    # User on compute instance created on demand
    user: user1

Example of an OpenStack deployment configuration file
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: YAML

  yorc:
    # Path to private key file on local host
    # used to connect to hosts on the bootstrapped setup
    private_key_file: /home/myuser/.ssh/yorc.pem
    # Path to Certificate Authority private key, accessible locally
    # If no key ile provided, one will be generated
    ca_key_file: /home/myuser//ca-key.pem
    # Certificate authority private key passphrase
    ca_passphrase: changeme
    # Path to PEM-encoded Certificate Authority, accessible locally
    # If not provided, a Certificate Authority will be generated
    ca_pem_file: /home/myuser/ca.pem
  locations:
    - name: firstOpenStackLocation
      type: openstack
      properties:
        auth_url: http://10.1.2.3:5000/v2.0
        default_security_groups:
        - secgroup1
        - secgroup2
        password: mypasswd
        private_network_name: private-test
        region: RegionOne
        tenant_name: mytenant
        user_name: myuser
  address:
    floating_network_name: mypublic-net
  compute:
    image: "7d9bd308-d9c1-4952-123-95b761672499"
    flavor: 3
    key_pair: yorc
  credentials:
    # User on compute instance created on demand
    user: user1


Example of a Hosts Pool deployment configuration file
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: YAML

  yorc:
    # Path to private key file on local host
    # used to connect to hosts on the bootstrapped setup
    private_key_file: /home/myuser/.ssh/yorc.pem
    # Path to Certificate Authority private key, accessible locally
    # If no key ile provided, one will be generated
    ca_key_file: /home/myuser//ca-key.pem
    # Certificate authority private key passphrase
    ca_passphrase: changeme
    # Path to PEM-encoded Certificate Authority, accessible locally
    # If not provided, a Certificate Authority will be generated
    ca_pem_file: /home/myuser/ca.pem
  compute:
    shareable: "false"
  hosts:
  - name: host1
    connection:
      user: user1
      host: 10.129.1.10
      port: 22
    labels:
      host.cpu_frequency: 3 GHz
      host.disk_size: 40 GB
      host.mem_size: 4GB
      host.num_cpus: "2"
      os.architecture: x86_64
      os.distribution: centos
      os.type: linux
      os.version: "7.3.1611"
      private_address: "10.0.0.10"
      public_address: "10.129.1.10"
  - name: host2
    connection:
      user: user1
      host: 10.129.1.11
      port: 22
    labels:
      environment: dev
      host.cpu_frequency: 3 GHz
      host.disk_size: 40 GB
      host.mem_size: 4GB
      host.num_cpus: "2"
      os.architecture: x86_64
      os.distribution: centos
      os.type: linux
      os.version: "7.3.1611"
      private_address: "10.0.0.11"
      public_address: "10.129.1.11"


Exporting and loading an interactive configuration file
-------------------------------------------------------

When deploying, the final configuration of the bootstrapping is automatically exported to a file. The name of the
file is the deployment id, which is a timestamp of current year to second. You can create a custom deployment id
using ''-n'' option :

.. parsed-literal::

    ./yorc bootstrap -n a_deploy_name

If you specify an already existing name (an input config file of the same name this already exists), an unique name will
be created, of the form ''nameN'', where N is an integer, generated incrementally.

You can then load a config file using the "-v" option :

.. parsed-literal::

    ./yorc bootstrap -v path_to_a_file_containing_input_values

Please note than if a config is loaded using this option, it will not be exported again.

If you wish to only export the interactive configuration without doing an actual bootstrap, just set the ''--config_only'' flag:

.. parsed-literal::

    ./yorc bootstrap --config_only

it will cause the yorc invocation to terminate straight after the export of interactive config.


Troubleshooting
===============

By default, debug logs are disabled. To enable them, you can export the environment
variable YORC_LOG and set it to ``1`` or ``DEBUG`` before starting the bootstrap:

.. parsed-literal::

    export YORC_LOG=1

Once the bootstrap deployment has started, the local yorc server logs are available
under ``<working dir>/yorc.log``, (<working dir> default value being the directory ``./work``).

To get the bootstrap deployment ID and current status, run :

.. parsed-literal::

    ./yorc deployments list

To follow deployment logs and see these logs from the beginning, run :

.. parsed-literal::

    ./yorc deployments logs <deployment ID> --from-beginning

When a deployment has failed, in addition to logs failure in the logs, you can
also get of summary of the deployment steps statuses to identify quickly which
step failed, running :

.. parsed-literal::

    ./yorc deployments info <deployment ID>

If a step failed on a transient error that is now addressed, it is possible to run
again manually the failed step, and resume the deployment running the following
commands.

First from the previous command ``./yorc deployments info <deployment ID>`` output,
you can find the task ID that failed.

You can now run this command to get the exact name of the step that failed :

.. parsed-literal::

    ./yorc deployments tasks info --steps <deployment ID> <task ID>

Identify the name of the step that failed.

Let's say for the example that it is the step ``TerraformRuntime_create`` which failed
on timeout downloading the Terraform distribution.

You can then go to the directory where you will find the ansible playbook corresponding to this step :

.. parsed-literal::

    cd <working directory>/deployments/<deployment ID>/ansible/<task ID>/TerraformRuntime/standard.create/

And from this directory, run again this step through this command:

.. parsed-literal::

    ansible-playbook -i hosts run.ansible.yml -v

If this manual execution was successful, you can mark the corresponding step as
fixed in the deployment, running :

.. parsed-literal::

    ./yorc deployments tasks fix <deployment ID> <task ID> TerraformRuntime

You can now resume the bootstrap deployment running :

.. parsed-literal::

    ./yorc deployments tasks resume <deployment ID>

