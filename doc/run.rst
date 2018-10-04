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

