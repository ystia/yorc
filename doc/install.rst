.. _yorc_install_section:

Install Yorc and requirements
==============================

Typical Yorc deployment for OpenStack
--------------------------------------

In order to provision softwares on virtual machines that do not necessary have a floating IP we recommend to install Yorc itself on a virtual machine 
in your OpenStack tenant. Alien4Cloud and the Alien4Cloud Yorc Plugin (see their dedicated documentation to know how to install them) may be collocated
on the same VM or resides in a different VM.

Virtual Machines created by Yorc should be connected to the same private network as the Yorc VM (the :ref:`--infrastructure_openstack_private_network_name <option_infra_os>`
configuration flag allows to do it automatically). In order to provision Floating IPs, this private network should be connected to the public network 
of the tenant through a router.


.. image:: _static/img/yorc-os-typical-deployment.png
   :align: center 
   :alt: Typical Yorc deployment for OpenStack
   :scale: 75%


Host requirements
-----------------

Yorc requires a Linux x86_64 system to operate with at least 2 CPU and 2 Go of RAM.

Packages installation
---------------------

Following packages are required to perform the installation:
  * python
  * python-pip
  * zip/unzip
  * openssh-client
  * wget 

Now you can proceed with the installation of softwares used by Yorc.

.. parsed-literal::

    sudo pip install ansible==\ |ansible_version|
    wget \https://releases.hashicorp.com/consul/\ |consul_version|\ /consul\_\ |consul_version|\ _linux_amd64.zip
    sudo unzip consul\_\ |consul_version|\ _linux_amd64.zip -d /usr/local/bin
    wget \https://releases.hashicorp.com/terraform/\ |terraform_version|\ /terraform\_\ |terraform_version|\ _linux_amd64.zip
    sudo unzip terraform\_\ |terraform_version|\ _linux_amd64.zip -d /usr/local/bin

Finally you can install the Yorc binary into ``/usr/local/bin``.

To support :ref:`Orchestrator-hosted operations <tosca_orchestrator_hosted_operations>` sandboxed into Docker containers the following
softwares should also be installed.

.. code-block:: bash

  # for apt based distributions
  sudo apt install Docker
  # for yum based distributions
  sudo yum install Docker
  # Docker should be running and configured to works with http proxies if any
  sudo systemctl enable docker
  sudo systemctl start docker
  
  sudo pip install docker-py

For a complete Ansible experience please install the following python libs:

.. code-block:: bash

  # To support json_query filter for jinja2
  sudo pip install jmespath
  # To works easily with CIDRs
  sudo pip install netaddr

To support Ansible SSH password authentication instead of common ssh keys, the sshpass helper program needs to be installed too.

.. code-block:: bash

  # for apt based distributions
  sudo apt install sshpass
  # for yum based distributions
  sudo yum install sshpass


Final setup
-----------

In order to provision softwares through ssh, you need to store the ssh private key that will be used to connect to the nodes under 
``$HOME/.ssh/yorc.pem`` where ``$HOME`` is the home directory of the user running Yorc. This key should part of the authorized keys on remote hosts.
Generally, for OpenStack, it corresponds to the private key of the keypair used to create the instance. 

.. note:: A common issue is to create a key file that does not comply the ssh requirements for private keys (should be readable by the user but not
          accessible by group/others read/write/execute).


