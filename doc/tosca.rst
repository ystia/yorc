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

Supported Operations implementations
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Currently Yorc supports as builtin implementations for operations:

* Bash scripts
* Python scripts
* Ansible Playbooks

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

In addition, any inputs parameters defined on the operation definition are also injected.

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



