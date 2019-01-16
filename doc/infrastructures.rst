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

Yorc Supported infrastructures
===============================

This section describes the state of our integration with supported infrastructures and their specificities

.. _yorc_infras_hostspool_section:

Hosts Pool
----------

.. only:: html

   |prod|

The Hosts Pool is a very special kind of infrastructure. It consists in registering existing Compute nodes into a pool managed by Yorc.
Those compute nodes could be physical or virtual machines, containers or whatever as long as Yorc can SSH into it. Yorc will be responsible to 
allocate and release hosts for deployments. This is safe to use it concurrently in a Yorc cluster, Yorc instances will synchronize amongst themselves to 
ensure consistency of the pool.  

To sum up this infrastructure type is really great when you want to use an infrastructure that is not yet supported by Yorc.
Just take care you're responsible for handling the compatibility or conflicts of what is already installed and what will be by Yorc on your hosts pool.
The best practice is using container isolation. This is especially true if a host can be shared by several apps by specifying in Tosca with the Compute **shareable** property.

Hosts management
~~~~~~~~~~~~~~~~

Yorc comes with a REST API that allows to manage hosts in the pool and to easily integrate it with other systems. The Yorc CLI leverage this REST API 
to make it user friendly, please refer to :ref:`yorc_cli_hostspool_section` for more information

Hosts Pool labels & filters
~~~~~~~~~~~~~~~~~~~~~~~~~~~

It is strongly recommended to associate labels to your hosts. Labels allow to filter hosts based on criteria. Labels are just a couple of key/value pair

.. _yorc_infras_hostspool_filters_section:

Filters Grammar
^^^^^^^^^^^^^^^

There are four kinds of filters supported by Yorc:

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
If those are specified in the topology, Yorc will automatically add a filter ``host.<property_name> >= <property_value> <property_unit>`` or ``os.<property_name> = <property_value>``
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
  * if present a label named ``public_address`` will be used as attribute ``public_address`` of the Compute.
  * if present, following labels will fill the ``networks`` attribute of the Compute node:

    * ``networks.<idx>.network_name`` (ie. ``networks.0.network_name``) 
    * ``networks.<idx>.network_id`` (ie. ``networks.0.network_id``) 
    * ``networks.<idx>.addresses`` as a coma separated list of addresses (ie. ``networks.0.addresses``)

The resources host pool labels (``host.num_cpus``, ``host.disk_size``, ``host.mem_size``) are automatically decreased and increased respectively when a host pool is allocated and released
only if you specify any of these Tosca ``host`` resources capabilities Compute in its Alien4Cloud applications.
If you apply a new configuration on allocated hosts with new host resources labels, they will be recalculated depending on existing allocations resources.

    

.. _yorc_infras_slurm_section:

Slurm
-----

.. only:: html

   |prod|

`Slurm <https://slurm.schedmd.com/>`_ is an open source, fault-tolerant, and highly scalable cluster management and job scheduling system for large and small Linux clusters.
It is wildly used in High Performance Computing and it is the default scheduler of the `Bull Super Computer Suite <https://atos.net/en/products/high-performance-computing-hpc>`_ .

Yorc interacts with Slurm to allocate nodes on its cluster but also to run Slurm jobs.

Jobs have been modeled in Tosca and this allows Yorc to execute them, either as simple jobs or as ``Singularity`` jobs.

`Singularity <https://www.sylabs.io/singularity/>`_ is a container system similar to Docker but designed to integrate well HPC environments. and let users execute a command inside a Singularity or Docker container as a job submission.
See `Working with jobs <https://yorc-a4c-plugin.readthedocs.io/en/latest/jobs.html>`_ for more information.

Yorc supports the following resources on Slurm:

  * Node Allocations as Computes
  * Jobs
  * Singularity Jobs.


Resources based scheduling
~~~~~~~~~~~~~~~~~~~~~~~~~~

TOSCA allows to specify `requirements on Compute nodes <http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/csd01/TOSCA-Simple-Profile-YAML-v1.2-csd01.html#DEFN_TYPE_CAPABILITIES_COMPUTE>`_
if specified ``num_cpus`` and  ``mem_size`` requirements are used to allocate only the required resoures on computes. This allows to share a Slurm managed compute
across several deployments. If not specified a whole compute node will be allocated.

Yorc also support `Slurm GRES <https://slurm.schedmd.com/gres.html>`_ based scheduling. This is generally used to request a host with a specific type of resource (consumable or not) 
such as GPUs.

.. _yorc_infras_google_section:

Google Cloud Platform
---------------------

.. only:: html

   |prod|

The Google Cloud Platform integration within Yorc is ready for production and we support the following resources:

  * Compute Instances
  * Persistent Disks
  * Virtual Private Clouds (VPC)
  * Static IP Addresses.

Future work
~~~~~~~~~~~

It is planned to support soon the following feature:

  * Cloud VPN

.. _yorc_infras_aws_section:

AWS
---

.. only:: html

   |dev|

The AWS integration within Yorc allows to provision:
  * EC2 Compute Instances.
  * Elastic IPs.

This part is ready for production but we plan to support soon the following features to make it production-ready:

  * Elastic Block Store provisioning
  * Networks provisioning with Virtual Private Cloud

Future work
~~~~~~~~~~~

  * We plan to work on modeling `AWS Batch Jobs <https://aws.amazon.com/batch/>`_ in TOSCA and execute them thanks to Yorc.
  * We plan to work on `AWS ECS <https://aws.amazon.com/ecs>`_ to deploy containers

.. _yorc_infras_openstack_section:

OpenStack
---------

.. only:: html

   |prod|

The `OpenStack <https://www.openstack.org/>`_ integration within Yorc is production-ready.
Yorc is currently supporting:

  * Compute Instances
  * Block Storages
  * Virtual Networks
  * Floating IPs provisioning.

Future work
~~~~~~~~~~~

  * We plan to work on modeling `OpenStack Mistral workflows <https://wiki.openstack.org/wiki/Mistral>`_ in TOSCA and execute them thanks to Yorc.
  * We plan to work on `OpenStack Zun <https://wiki.openstack.org/wiki/Zun>`_ to deploy containers directly on top of OpenStack

.. _yorc_infras_kubernetes_section:

Kubernetes
----------

.. only:: html
   
   |prod|

The `Kubernetes <https://kubernetes.io/>`_ integration within Yorc is now production-ready.
Yorc is currently supporting the following K8s resources:

  * Deployments.
  * Jobs.
  * Services.

The `Google Kubernetes Engine <https://cloud.google.com/kubernetes-engine/>`_ is also supported as a Kubernetes cluster.

Future work
~~~~~~~~~~~

It is planned to support soon the following features:

  * Persistent Volume Claims.
  * StatefulSets.

.. |prod| image:: https://img.shields.io/badge/stability-production%20ready-green.svg
.. |dev| image:: https://img.shields.io/badge/stability-stable%20but%20some%20features%20missing-yellow.svg
.. |incubation| image:: https://img.shields.io/badge/stability-incubating-orange.svg

