Janus Supported infrastructures
===============================

This section describes the state of our integration with supported infrastructures and their specificities

.. _janus_infras_hostspool_section:

Hosts Pool
----------

.. only:: html

   |prod|

The Hosts Pool is a very special kind of infrastructure. It consists in registering existing Compute nodes into a pool managed by Janus.
Those compute nodes could be physical or virtual machines, containers or whatever as long as Janus can SSH into it. Janus will be responsible to 
allocate and release hosts for deployments. This is safe to use it concurrently in a Janus cluster, Janus instances will synchronize amongst themselves to 
ensure consistency of the pool.  

To sum up this infrastructure type is really great when you want to use an infrastructure that is not yet supported by Janus.

Hosts management
~~~~~~~~~~~~~~~~

Janus comes with a REST API that allows to manage hosts in the pool and to easily integrate it with other systems. The Janus CLI leverage this REST API 
to make it user friendly, please refer to :ref:`janus_cli_hostspool_section` for more informations

Hosts Pool labels & filters
~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is strongly recommended to associate labels to your hosts. Labels allow to filter hosts based on criteria. Labels are just a couple of key/value pair

.. _janus_infras_hostspool_filters_section:

Filters Grammar
^^^^^^^^^^^^^^^

There are four kinds of filters supported by Janus:

  * Filters based on the presence of a label ``label_identifier`` will match if a label with the given name is associated with a host whatever its value is.
  * Filters based on equality to a value ``label_identifier (=|==|!=) value`` will match if the value associated with the given label is equals (``=`` and ``==``) or different (``!=``) to the given value
  * Filters based on sets ``label_identifier (in | not in) (value [, other_value])`` will match if the value associated with the given label is one (``in``) or is not one (``not in``) of the given values
  * Filters based on comparisons ``label_identifier (< | <= | > | >=) number[unit]`` will match if the value associated with the given label is a number and matches the comparison sign. A unit could be associated 
    with the number, currently supported units are golang durations ("ns", "us" , "ms", "s", "m" or "h"), bytes units ("B", "KiB", "KB", "MiB",	"MB", "GiB", "GB", "TiB", "TB", "PiB", "PB", "EiB", "EB") and
    `International System of Units (SI) <https://en.wikipedia.org/wiki/Metric_prefix>`_. The case of the unit does not matter.  

Here are some example:

  * ``gpu``
  * ``os.distribution != windows``
  * ``os.architecture == x86_64``
  * ``environment = "Q&A"``
  * ``environment in ( "Q&A", dev, edge)``
  * ``gpu.type not in (k20, m60)``
  * ``gpu_nb > 1``
  * ``os.mem_size >= 4 GB``
  * ``os.disk_size < 1tb``
  * ``max_allocation_time <= 120h``


Implicit filters & labels
^^^^^^^^^^^^^^^^^^^^^^^^^

TOSCA allows to specify `requirements on Compute hardware <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/csd01/TOSCA-Simple-Profile-YAML-v1.2-csd01.html#DEFN_TYPE_CAPABILITIES_COMPUTE>`_
and `Compute operating system <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/csd01/TOSCA-Simple-Profile-YAML-v1.2-csd01.html#DEFN_TYPE_CAPABILITIES_OPSYS>`_ .
These are capabilities named ``host`` and ``os`` in the `TOSCA node Compute <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/csd01/TOSCA-Simple-Profile-YAML-v1.2-csd01.html#DEFN_TYPE_NODES_COMPUTE>`_ .
If those are specified in the topology, Janus will automatically add a filter ``host.<property_name> >= <property_value> <property_unit>`` or ``os.<property_name> = <property_value>``
This will allow to select hosts matching the required criteria.

This means that it is strongly recommended to add the following labels to your hosts:
  * ``host.num_cpus``       (ie. host.num_cpus=4)
  * ``host.cpu_frequency``  (ie. host.cpu_frequency=3 GHz)
  * ``host.disk_size``      (ie. host.disk_size=50 GB)
  * ``host.mem_size``       (ie. host.mem_size=4GB)
  * ``os.architecture``     (ie. os.architecture=x86_64)
  * ``os.type``             (ie. os.type=linux)
  * ``os.distribution``     (ie. os.distribution=ubuntu)
  * ``os.version``          (ie. os.version=17.10)

Some labels are also automatically exposed as TOSCA Compute instance attributes:

  * if present a label named ``private_address`` will be used as attribute ``private_address`` and ``ip_address`` of the Compute. If not set the connection host will be used instead
    this allows to specify a network different for the applicative communication and for the orchestrator communication
  * if present, following labels will fill the ``networks`` attribute of the Compute node:

    * ``networks.<idx>.network_name`` (ie. ``networks.0.network_name``) 
    * ``networks.<idx>.network_id`` (ie. ``networks.0.network_id``) 
    * ``networks.<idx>.addresses`` as a coma separated list of addresses (ie. ``networks.0.addresses``)
    

.. _janus_infras_slurm_section:

Slurm
-----

.. only:: html

   |prod|

`Slurm <https://slurm.schedmd.com/>`_ is an open source, fault-tolerant, and highly scalable cluster management and job scheduling system for large and small Linux clusters.
It is wildly used in High Performance Computing and it is the default scheduler of the `Bull Super Computer Suite <https://atos.net/en/products/high-performance-computing-hpc>`_ .

Janus interacts with Slurm to allocate nodes on its cluster.

Resources based scheduling
~~~~~~~~~~~~~~~~~~~~~~~~~~

TOSCA allows to specify `requirements on Compute nodes <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/csd01/TOSCA-Simple-Profile-YAML-v1.2-csd01.html#DEFN_TYPE_CAPABILITIES_COMPUTE>`_
if specified ``num_cpus`` and  ``mem_size`` requirements are used to allocate only the required resoures on computes. This allows to share a Slurm managed compute
across several deployments. If not specified a whole compute node will be allocated.

Janus also support `Slurm GRES <https://slurm.schedmd.com/gres.html>`_ based scheduling. This is generally used to request a host with a specific type of resource (consumable or not) 
such as GPUs.

Future work
~~~~~~~~~~~

  * We plan to soon work on modeling Slurm Jobs in TOSCA and execute them thanks to Janus.
  * We also plan to support `Singularity <http://singularity.lbl.gov/>`_ , a container system similar to Docker but designed to integrate well HPC environments.
    This feature, as it will leverage some Bull HPC proprietary integration with Slurm, will be part of a premium version of Janus.

.. _janus_infras_aws_section:

AWS
---

.. only:: html

   |dev|

The AWS integration within Janus allows to provision Compute nodes and Elastic IPs on top of `AWS EC2 <https://aws.amazon.com/ec2/>`_ this part is ready for production
but we plan to support soon the following features to make it production-ready:

  * Support Elastic Block Store provisioning
  * Support Networks provisioning with Virtual Private Cloud

Future work
~~~~~~~~~~~

  * We plan to work on modeling `AWS Batch Jobs <https://aws.amazon.com/batch/>`_ in TOSCA and execute them thanks to Janus.
  * We plan to work on `AWS ECS <https://aws.amazon.com/ecs>`_ to deploy containers

.. _janus_infras_openstack_section:

OpenStack
---------

.. only:: html

   |prod|

The `OpenStack <https://www.openstack.org/>`_ integration within Janus is production-ready. We support Compute, Block Storage, Virtual Networks and Floating IPs
provisioning.

Future work
~~~~~~~~~~~

  * We plan to work on modeling `OpenStack Mistral workflows <https://wiki.openstack.org/wiki/Mistral>`_ in TOSCA and execute them thanks to Janus.
  * We plan to work on `OpenStack Zun <https://wiki.openstack.org/wiki/Zun>`_ to deploy containers directly on top of OpenStack

.. _janus_infras_kubernetes_section:

Kubernetes
----------

.. only:: html
   
   |incubation|

Kubernetes support is in a kind of Proof Of Concept phasis for now. We are currently working on a total refactoring of this part.

.. |prod| image:: https://img.shields.io/badge/stability-production%20ready-green.svg
.. |dev| image:: https://img.shields.io/badge/stability-stable%20but%20some%20features%20missing-yellow.svg
.. |incubation| image:: https://img.shields.io/badge/stability-incubating-orange.svg

