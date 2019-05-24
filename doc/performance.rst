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

.. _yorc_performance_section:

Performance
===========

.. _tosca_consul_performance_section:

Consul Storage
--------------

Yorc heavily relies on Consul to synchronize Yorc instances and store configurations, runtime data and TOSCA data models.
This lead to generate an important load pressure on Consul, under specific circumstances (thousands of deployments having
each an high number of TOSCA types and templates and poor latency performances on networks between Yorc and Consul server)
you may experience some performances issues specially during the initialization phase of a deployment. This is because we
this is when we store most of the data into Consul. To deal with this we recommend to carefully read
`the Consul documentation on performance <https://www.consul.io/docs/install/performance.html>`_ and update the default
configuration if needed.

You can find bellow some configuration options that could be adapted to fit your specific use case:

  * Yorc stores keys into Consul in highly parallel way, to prevent consuming too much connections and specially getting
    into a ``max open file descriptor issue`` we use a mechanism that limit the number of open connections to Consul.
    The number open connections can be set using :ref:`option_pub_routines_cmd`. The default value of ``500`` was determined
    empirically to fit the default ``1024`` maximum number of open file descriptors. Increasing this value could improve performances
    but you should also update accordingly the maximum number of open file descriptors.

  * Yorc 3.2 brings an experimental feature that allows to pack some keys storage into `Consul transactions <https://www.consul.io/api/txn.html>`_.
    This mode is not enabled by default and can be activated by setting the environment variable named ``YORC_CONSUL_STORE_TXN_TIMEOUT``
    to a valid `Go duration <https://golang.org/pkg/time/#ParseDuration>`_. Consul transactions can contain up to 64 operations,
    ``YORC_CONSUL_STORE_TXN_TIMEOUT`` defines a timeout after which we stop waiting for new operations to pack into a single transaction and submit
    it as it to Consul.
    We had pretty good results by setting this property to ``10ms``. This feature may became the default in the future after being tested
    against different use cases and getting some feedback from production deployments.

.. _tosca_operations_performance_section:

TOSCA Operations
----------------

As described in TOSCA :ref:`tosca_operations_implementations_section`, Yorc supports these builtin implementations for operations to execute on remote hosts :

  * Bash scripts
  * Python scripts
  * Ansible Playbooks

It is recommended to implement operations as Ansible Playbooks to get the best execution performance.

When operations are not implemented using Ansible playbooks, the following Yorc Server :ref:`yorc_config_file_ansible_section` settings allow to improve the performance of scripts execution on remote hosts :

  * ``use_openssh``: Prefer OpenSSH over Paramiko, a Python implementation of SSH (used by default) to provision remote hosts. OpenSSH have several optimization like reusing connections that should improve performance but may lead to issues on older systems
    See Ansible documentation on `Remote connection information <https://docs.ansible.com/ansible/latest/user_guide/intro_getting_started.html#remote-connection-information>`_.
  * ``cache_facts``: Caches `Ansible facts <https://docs.ansible.com/ansible/latest/user_guide/playbooks_variables.html#fact-caching>`_ (values fetched on remote hosts about network/hardware/OS/virtualization configuration) so that these facts are not recomputed each time a new operation is a run for a given deployment.
  * ``archive_artifacts``: Archives operation bash/python scripts locally, copies this archive and unarchives it on remote hosts (requires tar to be installed on remote hosts), to avoid multiple time consuming remote copy operations of individual scripts.
