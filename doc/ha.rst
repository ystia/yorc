Run Janus in High Availability (HA) mode
========================================

High level view of a typical HA installation
--------------------------------------------

The bellow figure illustrate how a typical Janus setup for enabling High Availability looks like.

.. image:: _static/img/Janus_HA.png
   :align: center 
   :alt: Typical Janus HA steup
   :scale: 75%


This setup is composed by the following main components:

  * A POSIX distributed file system (NFS as an example in the figure above) to store deployments recipes (Shell scripts, Ansible recipes, binaries...)
  * A cluster of Consul servers
  * A cluster of Janus servers each one collocated with a Consul agent and connected to the distributed filesystem
  * A Alien4Cloud with the Janus plugin collocated with a Consul agent

The next section describes how to setup those components.

Janus HA setup
--------------

Distributed File System
~~~~~~~~~~~~~~~~~~~~~~~

Describing how to setup a Distributed File System (DSF) is out of the scope of this document.
When choosing your DSF please take care to verify that it is POSIX compatible and can be mounted as linux partition.

Consul servers
~~~~~~~~~~~~~~

To setup a cluster of Consul servers please refer to the `Consul online documentation <https://www.consul.io/docs/guides/bootstrapping.html>`_.
One important thing to note is that you will need 3 or 5 Consul servers to ensure HA.

Janus servers
~~~~~~~~~~~~~

Each Janus server should be installed on is own host with a local Consul agent and a partition mounted on the Distributed File System.
Consul should run on agent mode (by opposition to the server mode) here is how to run a Consul in agent mode
and connect it to a running Consul server cluster.

.. code-block:: bash

    consul agent -config-dir ./consul-conf -data-dir ./consul-data -retry-join <ConsulServer1IP> -retry-join <ConsulServer2IP> -retry-join <ConsulServer3IP>

You should create a Consul service description for each Janus server and store it as a JSON file in the `consul-conf` directory

.. code-block:: json

    {
      "service": {
        "name": "janus",
        "tags": ["server", "{JANUS_ID}"],
        "address": "{PUBLIC_JANUS_IP}",
        "port": 8800,
        "check": {
          "name": "TCP check on port 8800",
          "tcp": "{PUBLIC_JANUS_IP}:8800",
          "interval": "10s"
        }
      }
    }


* Replace {JANUS_ID} with an unique ID for each Janus instance (e.g. "server1" and "server2").
* Replace {PUBLIC_JANUS_IP} with current Janus IP used by Alien4Cloud to contact Janus.

When running Janus you should use the :ref:`--working_directory <option_workdir_cmd>` command line flag 
(or equivalent configuration options or environment variable) to specify a working directory on the 
Distributed File System.

Alien4Cloud
~~~~~~~~~~~

Please refer to the dedicated Janus plugin for Alien4Cloud documentation for its typical installation and configuration.

Install and run Consul in agent mode.

.. code-block:: bash

    consul agent -config-dir ./consul-conf -data-dir ./consul-data -retry-join <ConsulServer1IP> -retry-join <ConsulServer2IP> -retry-join <ConsulServer3IP> -recursor <ConsulServer1IP> -recursor <ConsulServer2IP> -recursor <ConsulServer3IP>

`Configure Consul DNS forwarding <https://www.consul.io/docs/guides/forwarding.html>`_ in order to be able to resolve `janus.service.consul` DNS domain name.

In the Janus plugin for Alien4Cloud configuration use `http://janus.service.starlings:8800` as Janus URL instead of using a IP address.
This DNS name will be resolved by Consul (using a round-robin algorithm) to available Janus servers.

If a Janus server become unavailable then Consul will detect it by using the service check and will stop to resolve the DNS request to this Janus instance, allowing seamless failover.


