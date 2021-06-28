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

TOSCA support in Yorc
======================

TOSCA stands for Topology and Orchestration Specification for Cloud Applications. It is an 
`OASIS <https://www.oasis-open.org/>`_ standard specification. Currently Yorc implements the version
`TOSCA Simple Profile in YAML Version 1.2 <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html>`_ 
of this specification.

Yorc is a TOSCA based orchestrator, meaning that it consumes TOSCA definitions and services templates to perform applications deployments 
and lifecycle management. 

Yorc is also workflow driven meaning that it will execute workflows defined in a TOSCA service template to perform a deployment.
The easiest way to generate valid TOSCA definitions for Yorc is to use a TOSCA web composer called `Alien4Cloud <http://alien4cloud.github.io/>`_.

Yorc provides an Alien4Cloud (A4C) plugin that allows A4C to interact with Yorc.

Alien4Cloud provides `a great documentation on writing TOSCA components <http://alien4cloud.github.io/#/documentation/1.4.0/devops_guide/dev_ops_guide.html>`_.

Bellow are the specificities of Yorc

TOSCA Operations
----------------

Supported TOSCA functions
~~~~~~~~~~~~~~~~~~~~~~~~~

Yorc supports the following functions that could be used in value assignments (generally for attributes and operation inputs).

- ``get_input: [input_name]``: Will retrieve the value of a Topology's input 
- ``get_property: [<entity_name>, <optional_cap_name>, <property_name>, <nested_property_name_or_index_1>, ..., <nested_property_name_or_index_n> ]``: Will retrieve 
  the value of a property in a given entity. ``<entity_name>`` could be the name of a given node or relationship template, ``SELF`` for the entity holding this function,
  ``HOST`` for one of the hosts (in the hosted-on hierarchy) of the entity holding this function, ``SOURCE``, ``TARGET`` respectively for the source or the target entity
  in case of a relationship. ``<optional_cap_name>`` is optional and allows to specify that we target a property in a capability rather directly on the node.
- ``get_attribute: [<entity_name>, <optional_cap_name>, <property_name>, <nested_property_name_or_index_1>, ..., <nested_property_name_or_index_n> ]``: see ``get_property`` above
- ``concat: [<string_value_expressions_*>]``: concats the result of each nested expression. Ex: ``concat: [ "http://", get_attribute: [ SELF, public_address ], ":", get_attribute: [ SELF, port ] ]``
- ``get_operation_output: [<modelable_entity_name>, <interface_name>, <operation_name>, <output_variable_name>]``: Retrieves the output of an operation
- ``get_secret: [<secret_path>, <optional_implementation_specific_options>]``: instructs to look for the value within a connected vault instead of within the Topology. Resulting value is considered as a secret by Yorc.

.. _tosca_operations_implementations_section:

Supported Operations implementations
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Currently Yorc supports as builtin implementations for operations:

* Bash scripts
* Python scripts
* Ansible Playbooks

Basic support for raws scripts execution through Ansible playbooks is also provided on Windows targets configured to use ssh.

New implementations can be plugged into Yorc using its plugin mechanism.

.. todo:
    Document the plugin mechanism and reference it here

Execution Context
~~~~~~~~~~~~~~~~~

Python and Bash scripts are executed by a wrapper script used to retrieve operations outputs. This script itself is executed using
a ``bash -l`` command meaning that the login profile of the user used to connect to the host will be loaded.

.. warning::

    Defining operations inputs with the same name as Bash reserved variables like ``USER``, ``HOME``, ``HOSTNAME`` `and so on... <http://tldp.org/LDP/abs/html/internalvariables.html>`_ 
    may lead to unexpected results... Avoid to use them.  

Injected Environment Variables
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

When operation scripts are called, some environment variables are injected by Yorc.

- For Python and Bash scripts those variables are injected as environment variables.
- For Python scripts they are also injected as global variables of the script and can be used directly. 
- For Ansible playbooks they are injected as `Playbook variables <http://docs.ansible.com/ansible/latest/playbooks_variables.html>`_.

Operation outputs
~~~~~~~~~~~~~~~~~

TOSCA `defines a function called get_operation_output <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/csd01/TOSCA-Simple-Profile-YAML-v1.2-csd01.html#DEFN_FUNCTION_GET_OPERATION_OUTPUT>`_,
this function instructs Yorc to retrieve a value at the end of a operation. In order to allow Yorc to retrieve those values you should depending on your operation 
implementation:

* in Bash scripts you should export a variable named as the output variable (case sensitively)
* in Python scripts you should define a variable (globally to your script root not locally to a class or function) named as the output variable (case sensitively)
* in Ansible playbooks you should set a fact named as the output variable (case sensitively)

Node operation
^^^^^^^^^^^^^^
For node operation script, the following variables are available:

* NODE: the node name.
* INSTANCE: the unique instance ID.
* INSTANCES: A comma separated list of all available instance IDs.
* HOST: the node name of the node that host the current one.
* DEPLOYMENT_ID: the unique deployment identifier.

In addition, any inputs parameters defined on the operation definition are also injected.


Relationship operation
^^^^^^^^^^^^^^^^^^^^^^

For relationship operation script, the following variables are available:

* TARGET_NODE: The node name that is targeted by the relationship.
* TARGET_INSTANCE: The instance ID that is targeted by the relatonship.
* TARGET_INSTANCES: Comma separated list of all available instance IDs for the target node.
* TARGET_HOST: The node name of the node that host the node that is targeted by the relationship.
* SOURCE_NODE: The node name that is the source of the relationship.
* SOURCE_INSTANCE: The instance ID of the source of the relationship.
* SOURCE_INSTANCES: Comma separated list of all available source instance IDs.
* SOURCE_HOST: The node name of the node that host the node that is the source of the relationship.
* DEPLOYMENT_ID: the unique deployment identifier.

In addition, properties and attributes of the target capability of the relationship are injected automatically.
We do this as they can only be retrieved by knowing in advance the actual name of the capability, which is not
very practical when designing a generic operation. As a target component may have several capabilities that match
the relationship capability type we inject the following variables:

* TARGET_CAPABILITY_NAMES: comma-separated list of matching capabilities names. It could be used to loop over the injected variables
* TARGET_CAPABILITY_<capabilityName>_TYPE: actual type of the capability
* TARGET_CAPABILITY_TYPE: actual type of the capability of the first matching capability
* TARGET_CAPABILITY_<capabilityName>_PROPERTY_<propertyName>: value of a property
* TARGET_CAPABILITY_PROPERTY_<propertyName>: value of a property for the first matching capability
* TARGET_CAPABILITY_<capabilityName>_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance
* TARGET_CAPABILITY_<instanceName>_ATTRIBUTE_<attributeName>: value of an attribute of a given instance for the first matching capability

Finally, any inputs parameters defined on the operation definition are also injected.

Attribute and multiple instances
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

When an operation defines an input, the value is available by fetching an environment variable. If you have multiple instances,
 you’ll be able to fetch the input value for all instances by prefixing the input name by the instance ID.

Let’s imagine you have an relationship’s configure interface operation defined like this:

.. code-block:: YAML

    add_target:
      inputs:
        TARGET_IP: { get_attribute: [TARGET, ip_address] }
      implementation: scripts/add_target.sh


Let’s imagine we have a node named MyNodeS with 2 instances: MyNodeS_1, MyNodeS_2. The node MyNodeS is connected to the target 
node MyNodeT which has also 2 instances MyNodeT_1 and MyNodeT_2.

When the add_target.sh script is executed for the relationship instance that connects MyNodeS_1 to MyNodeT_1, the following 
variables will be available:

.. code-block:: bash

    TARGET_NODE=MyNodeT
    TARGET_INSTANCE=MyNodeT_1
    TARGET_INSTANCES=MyNodeT_1,MyNodeT_2
    SOURCE_NODE=MyNodeS
    SOURCE_INSTANCE=MyNodeS_1
    SOURCE_INSTANCES=MyNodeS_1,MyNodeS_2
    TARGET_IP=192.168.0.11
    MyNodeT_1_TARGET_IP=192.168.0.11
    MyNodeT_2_TARGET_IP=192.168.0.12

.. _tosca_orchestrator_hosted_operations:

Orchestrator-hosted Operations
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In the general case an operation is an implementation of a step within a node's lifecycle
(install a software package for instance). Those operations should be executed on the Compute that hosts 
the node. Yorc handles this case seamlessly and execute your implementation artifacts on the required host.

But sometimes you may want to model in TOSCA an interaction with something (generally a service) that is 
not hosted on a compute of your application.
For those usecases the TOSCA specification support a tag called *operation_host* this tag could be set either
on `an operation implementation <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ELEMENT_OPERATION_DEF>`_  
or on `a workflow step <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_WORKFLOW_STEP_DEFN>`_.
If set to the keyword ``ORCHESTRATOR`` this tag indicates that the operation should be executed on the host of the 
orchestrator.

For executing those kind of operations Yorc supports two different behaviors. The first one is to execute implementation
artifacts directly on the orchestrator's host. But we think that running user-defined bash or python scripts
directly on the orchestrator's host may be dangerous. So, Yorc offers an alternative that allows to run those
scripts in a sandboxed environment implemented by a Docker container. This is the recommended solution.

Choosing one or the other solution is done by configuration see 
:ref:`ansible hosted operations options in the configuration section <option_ansible_sandbox_hosted_ops_cfg>`.
If a :ref:`default_sandbox <option_ansible_sandbox_hosted_ops_default_sandbox_cfg>` option is provided, it
will be used to start a docker sandbox. Otherwise if 
:ref:`unsandboxed_operations_allowed <option_ansible_sandbox_hosted_ops_unsandboxed_flag_cfg>` is set to ``true``
(defaults to ``false``) then operations are executed on orchestrator's host. Otherwise Yorc will rise an
error if an orchestrator hosted operation should be executed.

In order to let Yorc interact with Docker to manage sandboxes some requirements should be met on the Yorc's host:

  * Docker service should be installed and running
  * Docker CLI should be installed
  * the *pip package* ``docker_py`` should be installed

Yorc uses standard Docker's APIs so ``DOCKER_HOST`` and ``DOCKER_CERT_PATH`` environment variables could be used
to configure the way Yorc interacts with Docker.

To execute operations on containers, Ansible will by default detect the python interpreter to use
as described in `Ansible documentation <https://docs.ansible.com/ansible/latest/reference_appendices/interpreter_discovery.html>`_ section ``auto_silent``.
To provide yourself a python interpreter path, use the Ansible behavioral inventory parameter ``ansible_python_interpreter``, like below in a Yaml
Yorc configuration excerpt specifying to use python3 :

.. code-block:: YAML

  ansible:
    inventory:
      "hosted_operations:vars":
      - ansible_python_interpreter=/usr/bin/python3

See :ref:`Ansible Inventory Configuration section <option_ansible_inventory_cfg>`
for more details.

Apart from the requirements described above, you can install whatever you want
in your Docker image as prerequisites of your operations artifacts.

Yorc will automatically pull the required Docker image and start a separated Docker sandbox before each 
orchestrator-hosted operation and automatically destroy it after the operation execution.  

.. caution:: Currently setting ``operation_host`` on operation implementation is supported in Yorc but not in Alien4Cloud.
             That said, when using Alien4Cloud workflows will automatically be generated with ``operation_host=ORCHESTRATOR``
             for nodes that are not hosted on a Compute.

