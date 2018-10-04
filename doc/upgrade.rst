.. _yorc_upgrades_section:

Upgrades
========

Upgrades of Yorc that preserve already deployed applications are available starting with Yorc 3.1.0.
It is safe to upgrade from Yorc 3.0.0 but upgrading from previous releases can lead to unpredictable results.

An upgrade leads to a service interruption. Currently the standard process is to stop all running instances of Yorc.
Upgrade dependencies like Consul, Terraform, Ansible for instance, then upgrade Yorc itself and restart it.
Yorc will automatically take care of upgrading its database schema by its own from version 3.0.0 up to its
current version.

.. note:: A rolling upgrade without interruption feature is planned for future versions.

.. _yorc_upgrades_310_section:

Upgrading to Yorc 3.1.0
-----------------------

Ansible
~~~~~~~

Ansible needs to be upgraded to version 2.6.3, run the following command to
do it:

.. code-block:: bash

    sudo pip install ansible==2.6.3

Terraform
~~~~~~~~~

Terraform needs to be upgraded to version 0.11.8. Moreover this version comes
with a new packaging where providers are not shipped anymore in the main
binary. So you also need to download them separately.


.. code-block:: bash

    # Install the new Terraform version
    wget https://releases.hashicorp.com/terraform/0.11.8/terraform_0.11.8_linux_amd64.zip
    sudo unzip terraform_0.11.8_linux_amd64.zip -d /usr/local/bin

    # Now install Terraform plugins
    sudo mkdir -p /var/terraform/plugins

    wget https://releases.hashicorp.com/terraform-provider-consul/2.1.0/terraform-provider-consul_2.1.0_linux_amd64.zip
    sudo unzip terraform-provider-consul_2.1.0_linux_amd64.zip -d /var/terraform/plugins

    wget https://releases.hashicorp.com/terraform-provider-null/1.0.0/terraform-provider-null_1.0.0_linux_amd64.zip
    sudo unzip terraform-provider-null_1.0.0_linux_amd64.zip -d /var/terraform/plugins

    wget https://releases.hashicorp.com/terraform-provider-aws/1.36.0/terraform-provider-aws_1.36.0_linux_amd64.zip
    sudo unzip terraform-provider-aws_1.36.0_linux_amd64.zip -d /var/terraform/plugins

    wget https://releases.hashicorp.com/terraform-provider-google/1.18.0/terraform-provider-google_1.18.0_linux_amd64.zip
    sudo unzip terraform-provider-google_1.18.0_linux_amd64.zip -d /var/terraform/plugins

    wget https://releases.hashicorp.com/terraform-provider-openstack/1.9.0/terraform-provider-openstack_1.9.0_linux_amd64.zip
    sudo unzip terraform-provider-openstack_1.9.0_linux_amd64.zip -d /var/terraform/plugins

    sudo chmod 775 /var/terraform/plugins/*

Consul
~~~~~~

Ansible needs to be upgraded to version 1.2.3, run the following command to
do it:

.. code-block:: bash

    wget https://releases.hashicorp.com/consul/1.2.3/consul_1.2.3_linux_amd64.zip
    sudo unzip consul_1.2.3_linux_amd64.zip -d /usr/local/bin


Then restart Consul.

The recommended way to upgrade Consul is to perform a rolling upgrade.
See `Consul documentation <https://www.consul.io/docs/upgrading.html>`_ for details.
