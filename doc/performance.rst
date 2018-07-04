.. _yorc_performance_section:

Performance
===========

.. _tosca_operations_performance_section:

TOSCA Operations
----------------

As described in TOSCA :ref:`tosca_operations_implementations_section`, Yorc supports these builtin implementations for operations to execute on remote hosts :

  * Bash scripts
  * Python scripts
  * Ansible Playbooks

It is recommended to implement operations as Ansible Playbooks to get the best execution performance.

When operations are not implemented using Ansible playbooks, the following Yorc Server :ref:`yorc_config_file_ansible_section` settings allow to improve the performance of scripts execution on remote hosts :

  * ``use_openssh``: Prefer OpenSSH over Paramiko, a Python implementation of SSH (used by default) to provision remote hosts. OpenSSH have several optimization like reusing connections that should improve preformance but may lead to issues on older systems
    See Ansible documentation on `Remote connection information <https://docs.ansible.com/ansible/latest/user_guide/intro_getting_started.html#remote-connection-information>`_.
  * ``cache_facts``: Caches `Ansible facts <https://docs.ansible.com/ansible/latest/user_guide/playbooks_variables.html#fact-caching>`_ (values fetched on remote hosts about network/hardware/OS/virtualization configuration) so that these facts are not recomputed each time a new operation is a run for a given deployment.
  * ``archive_artifacts``: Archives operation bash/python scripts locally, copies this archive and unarchives it on remote hosts (requires tar to be installed on remote hosts), to avoid multiple time consuming remote copy operations of individual scripts.
