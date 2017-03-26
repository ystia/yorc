Install Janus and requirements
==============================

Host requirements
-----------------

Janus requires a Linux x86_64 system to operate with at least 2 CPU and 2 Go of RAM.

Packages installation
---------------------

Following packages are required to perform the installation:
  * python
  * pip
  * zip/unzip
  * openssh-client
  * wget 

Now you can proceed with the installation of softwares used by Janus.

.. code-block:: bash

    sudo pip install ansible==2.1.4.0
    wget https://releases.hashicorp.com/consul/0.7.5/consul_0.7.5_linux_amd64.zip
    sudo unzip consul_0.7.5_linux_amd64.zip -d /usr/local/bin
    wget https://releases.hashicorp.com/terraform/0.7.3/terraform_0.7.3_linux_amd64.zip
    sudo unzip terraform_0.7.3_linux_amd64.zip -d /usr/local/bin

Finally you can install the Janus binary into ``/usr/local/bin``.

Final setup
-----------

In order to provision softwares through ssh (Ansible), you need to store the ssh private key that will be used to connect to the nodes under 
``$HOME/.ssh/janus.pem`` where ``$HOME`` is the home directory of the user running Janus. This key should part of the authorized keys on remote hosts.
Generally, for OpenStack, it corresponds to the private key of the keypair used to create the instance. 

.. note:: A common issue is to create a key file that does not comply the ssh requirements for private keys (should be readable by the user but not
          accessible by group/others read/write/execute).


