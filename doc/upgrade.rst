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

.. _yorc_upgrades_section:

Upgrades
========

Upgrades of Yorc that preserve already deployed applications are available starting with Yorc 3.1.0.
It is safe to upgrade from Yorc 3.0.0 but upgrading from previous releases can lead to unpredictable results.

An upgrade leads to a service interruption. Currently the standard process is to stop all running instances of Yorc.
Upgrade dependencies like Consul, Terraform, Ansible for instance, then upgrade Yorc itself and restart it.
Yorc will automatically take care of upgrading its database schema by its own from version 3.0.0 up to its
current version.

By default Yorc takes a snapshot of the Consul database before upgrading and automatically rollback to this snapshot
if an error occurs during the upgrade process. If you are running Consul with ACL enabled the snapshot and restore
feature requires to have the ``management`` ACL. It is possible to disable this feature by setting the
``YORC_DISABLE_CONSUL_SNAPSHOTS_ON_UPGRADE`` environment variable to ``1`` or ``true``.

.. note:: A rolling upgrade without interruption feature is planned for future versions.

.. _yorc_upgrades_320_section:

Upgrading to Yorc 3.2.0
-----------------------

Ansible
~~~~~~~

Although Yorc 3.2.0 can still work with Ansible 2.7.2, security vulnerabilities were
identified in this Ansible version.

So, it is strongly advised to upgrade Ansible to version 2.7.9:

.. code-block:: bash

    sudo pip install ansible==2.7.9

.. _yorc_upgrades_310_section:

Upgrading to Yorc 3.1.0
-----------------------

Ansible
~~~~~~~

Ansible needs to be upgraded to version 2.7.2, run the following command to
do it:

.. code-block:: bash

    sudo pip install ansible==2.7.2

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

Consul needs to be upgraded to version 1.2.3, run the following command to
do it:

.. code-block:: bash

    wget https://releases.hashicorp.com/consul/1.2.3/consul_1.2.3_linux_amd64.zip
    sudo unzip consul_1.2.3_linux_amd64.zip -d /usr/local/bin


Then restart Consul.

The recommended way to upgrade Consul is to perform a rolling upgrade.
See `Consul documentation <https://www.consul.io/docs/upgrading.html>`_ for details.
