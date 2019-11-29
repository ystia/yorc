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

.. _yorc_docker_section:

Run Yorc in a docker container
==============================

Along with a Yorc release we also provide a docker image that ships Yorc and its dependencies.

This docker image is published in several ways:

  * A ``tgz`` version of the image is published on the `github release page <https://github.com/ystia/yorc/releases>`_ for each release.

  * Pre-release (milestone) versions and development branches are published in `Artifactory <https://ystia.jfrog.io/ystia/webapp/#/artifacts/browse/tree/General/docker>`_.
    You can use it to pull the image directly from the docker command.
    For instance: ``docker pull ystia-docker.jfrog.io/ystia/yorc:3.0.0-M5``

  * Starting with Yorc 3.0 GA version are also published on `the docker hub <https://hub.docker.com/r/ystia/yorc/>`_
    You can use it to pull the image directly from the docker command.
    For instance: ``docker pull ystia/yorc:3.0.0``

Image components
----------------

This Docker image is made of the following components:

S6 init system
~~~~~~~~~~~~~~

We use the `S6 overlay for containers <https://github.com/just-containers/s6-overlay>`_ in order to
have a minimal init system that supervise our services and
`reap zombies processes <https://blog.phusion.nl/2015/01/20/docker-and-the-pid-1-zombie-reaping-problem/>`_.

Consul
~~~~~~

`Consul <https://www.consul.io/>`_ is the distributed data store for Yorc. By default it will run in an isolated server mode
to provide a fully functional container out of the box. But it could be configured to connect
to a server and run in agent mode.

The Consul binary is installed in this image as described in the
:ref:`Yorc install section<yorc_install_section>`. It is run as a user and group named ``consul``.
Consul data directory is set by default to ``/var/consul``.

Configure Consul
^^^^^^^^^^^^^^^^

To configure Consul in the container you can either mount configuration files into ``/etc/consul``
or use environment variables. Following special variables are recognized:

  * ``CONSUL_MODE``: if set to ``server`` or not defined then Consul will run in server mode. Any other configuration
    will lead to Consul running in agent mode.

  * ``NB_CONSUL_SERVER``: allows to set the `bootstrap-expect <https://www.consul.io/docs/agent/options.html#_bootstrap_expect>`_
    command line flag for consul. If ``CONSUL_MODE`` is ``server`` and ``NB_CONSUL_SERVER`` is not defined then it defaults to ``1``.

  * ``SERVERS_TO_JOIN``: allows to provide a coma-separated list of server to connects to. This works either in server or agent mode.

In addition any environment variable that starts with ``CONSUL_ENV_`` will be added to a dedicated consul configuration file.
The format is ``CONSUL_ENV_<option_name>=<config_snippet>``. Here are some examples to make it clear:

``docker run -e 'CONSUL_ENV_ui=true' -e 'CONSUL_ENV_watches=[{"type":"checks","handler":"/usr/bin/health-check-handler.sh"}]' -e 'CONSUL_ENV_datacenter="east-aws"' ystia/yorc``

Will result in the following configuration file:

.. code-block:: JSON

  {
    "ui": true,
    "watches": [{"type":"checks","handler":"/usr/bin/health-check-handler.sh"}],
    "datacenter": "east-aws"
  }

go-dnsmasq
~~~~~~~~~~

go-dnsmasq is a lightweight DNS caching server/forwarder with minimal filesystem and runtime overhead.
It is used in this image to forward any ``*.consul`` dns request directly to Consul and forward others
dns requests to your standard upstream dns server. This allows to support dns resolving of Consul
services out of the box.

Ansible & Terraform
~~~~~~~~~~~~~~~~~~~

`Ansible <https://www.ansible.com/>`_ and `Terraform <https://www.terraform.io/>`_ are installed in this image as described in the
:ref:`Yorc install section<yorc_install_section>`.

There is no specific configuration needed for those components.

Docker
~~~~~~

The Docker client binary and the ``docker-py`` python library are installed in this image as described in the
:ref:`Yorc install section<yorc_install_section>`.

They are necessary to support :ref:`Orchestrator-hosted operations <tosca_orchestrator_hosted_operations>`
isolated in a Docker sandbox.

In order to let Yorc run Docker containers you should either expose the Docker service of your host in TCP
and configure Yorc to use this endpoint or mount the Docker socket into the container (recommended).

Here is the command line that allows to mount the Docker socket into the Yorc container:

.. code-block:: BASH

  # Using the --mount flag (recommended way on Docker 17.06+)
  docker run --mount "type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock" ystia/yorc
  # Using the -v flag (for Docker < 17.06)
  docker run -v /var/run/docker.sock:/var/run/docker.sock ystia/yorc

Yorc
~~~~

The Yorc binary is installed in this image as described in the
:ref:`Yorc install section <yorc_install_section>`.

Yorc is run as a ``yorc`` (group ``yorc``) user. This user's home directory is
``/var/yorc`` and the yorc process is run within that directory. Yorc's plugins can be
added using a mount within the ``/var/yorc/plugins`` directory.

Configuring Yorc
^^^^^^^^^^^^^^^^

To configure Yorc you can either mount a ``config.yorc.[json|yaml]`` into the ``/etc/yorc``
directory or use Yorc standard environment variables (for both cases see :ref:`yorc_config_section` section)
