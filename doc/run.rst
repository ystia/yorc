Starting Janus
==============

Starting Consul
---------------

Janus requires a running Consul instance prior to be started.

Here is how to start a standalone Consul server instance on the same host than Janus:

.. code-block:: bash

    consul agent -server -bootstrap-expect 1 -data-dir ./consul-data

.. note:: Wait for the ``agent: Synced service 'consul'`` log message to appear before continuing

Starting Janus
--------------

Please report to the :ref:`janus_config_section` for an exhaustive list of Janus' configuration options.
At least OpenStack access configuration files should be provided either by command-line flags, environment variables or configuration elements.
They are omitted bellow for brevity and considered as provided by a configuration file in one of the default location.

.. code-block:: bash

    janus server

