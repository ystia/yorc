Starting Yorc
==============

Starting Consul
---------------

Yorc requires a running Consul instance prior to be started.

Here is how to start a standalone Consul server instance on the same host than Yorc:

.. code-block:: bash

    consul agent -server -bootstrap-expect 1 -data-dir ./consul-data

.. note:: Wait for the ``agent: Synced service 'consul'`` log message to appear before continuing

Starting Yorc
--------------

Please report to the :ref:`yorc_config_section` for an exhaustive list of Yorc' configuration options.
At least OpenStack access configuration files should be provided either by command-line flags, environment variables or configuration elements.
They are omitted bellow for brevity and considered as provided by a configuration file in one of the default location.

Note that if you are using a passphrase on your ssh key, you have to start an ssh-agent before launching yorc. It is strongly recommended to start one by giving him a socket name.

.. code-block:: bash

    eval `ssh-agent -a /tmp/ssh-sock`

So in case of your ssh-agent process die, just restart it with the command above.

If your ssh key does not have a passphrase, **do not start any ssh-agent** before starting yorc and make sure that environement variable SSH_AUTH_SOCK is not set.

.. code-block:: bash

    killall ssh-agent
    unset SSH_AUTH_SOCK 

Then start yorc

.. code-block:: bash

    yorc server

