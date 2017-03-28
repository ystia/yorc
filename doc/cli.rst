Janus Command Line Interface
============================

You can interact with a Janus server using a command line interface (CLI). The same binary as for running a Janus server is used for the CLI.

General Options
---------------

  * ``--janus-api``: Specifies the host and port used to join the Janus' REST API. Defaults to ``localhost:8800``. Configuration entry ``janus_api`` and env var ``JANUS_API`` may also be used.
  * ``--no-color``: Disable coloring output (By default coloring is enable). 
  * ``-s`` or ``--secured``: Use HTTPS to connect to the Janus REST API
  * ``--ca-file``: This provides a file path to a PEM-encoded certificate authority. This implies the use of HTTPS to connect to the Janus REST API.
  * ``--skip-tls-verify``: skip-tls-verify controls whether a client verifies the server's certificate chain and host name. If set to true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing. This implies the use of HTTPS to connect to the Janus REST API.

CLI Commands related to deployments
-----------------------------------

All deployments related commands are sub-commands of a command named ``deployments``. 
In practice that means that the commands starts with 

.. code-block:: bash
    
    janus deployments

For brevity ``deployments`` supports the following aliases: ``depls``, ``depl``, ``deps``, ``dep`` and ``d``.

Deploy a CSAR
~~~~~~~~~~~~~

Deploys a file or directory pointed by <csar_path>
If <csar_path> point to a valid zip archive it is submitted to Janus as it.
If <csar_path> point to a file or directory it is zipped before beeing submitted to Janus.
If <csar_path> point to a single file it should be TOSCA YAML description.

.. code-block:: bash

     janus deployments deploy <csar_path> [flags]
     
Flags:
  * ``-e``, ``--stream-events``: Stream events after deploying the CSAR.
  * ``-l``, ``--stream-logs``: Stream logs after deploying the CSAR. In this mode logs can't be filtered, to use this feature see the "log" command.

Undeploy a deployment
~~~~~~~~~~~~~~~~~~~~~

Undeploy an application specifying the deployment ID.

.. code-block:: bash

     janus deployments undeploy <DeploymentId> [flags]
     
Flags:
  * ``-p``, ``--purge``: To use if you want to purge instead of undeploy.
  * ``-e``, ``--stream-events``: Stream events after deploying the CSAR.
  * ``-l``, ``--stream-logs``: Stream logs after deploying the CSAR. In this mode logs can't be filtered, to use this feature see the "log" command.


List deployments
~~~~~~~~~~~~~~~~

List active deployments. Giving there ids and statuses.

.. code-block:: bash

    janus deployments list


Get information on a specific deployment
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Display information about a given deployment.
It prints the deployment status and the status of all the nodes contained in this deployment.

.. code-block:: bash

     janus deployments info <DeploymentId> [flags]
     
Flags:
  * ``-d``, ``--detailed``: Add details to the info command making it less concise and readable.

Get deployment events
~~~~~~~~~~~~~~~~~~~~~

Streams events for a given deployment id

.. code-block:: bash

     janus deployments events <DeploymentId> [flags]
     
Flags:
  * ``-b``, ``--from-beginning``: Show events from the beginning of a deployment
  * ``-n``, ``--no-stream``: Show events then exit. Do not stream events. It implies --from-beginning

Get deployment logs
~~~~~~~~~~~~~~~~~~~

Streams logs for a given deployment id

.. code-block:: bash

     janus deployments logs <DeploymentId> [flags]
     
Flags:
  * ``-f``, ``--filter``: Allows to filters logs by type. Accepted filters are "engine" for Janus logs, "infrastructure" for infrastructure 
    provisioning logs or "software" for software provisioning. This flag may appear several times and may contain a coma separated list of filters.
    If not specified logs are not filtered.
  * ``-b``, ``--from-beginning``: Show logs from the beginning of a deployment
  * ``-n``, ``--no-stream``: Show logs then exit. Do not stream logs. It implies --from-beginning

Scale a specific node
~~~~~~~~~~~~~~~~~~~~~

Scale a given node of a deployment <DeploymentId> by adding or removing the specified number of instances.

.. code-block:: bash

     janus deployments scale <DeploymentId> [flags]

Flags:
  * ``-d``, ``--delta``: The non-zero number of instance to add (if > 0) or remove (if < 0).
  * ``-n``, ``--node``: The name of the node that should be scaled.
  * ``-e``, ``--stream-events``: Stream events after  issuing the scaling request.
  * ``-l``, ``--stream-logs``: Stream logs after issuing the scaling request. In this mode logs can't be filtered, to use this feature see the "log" command.

Execute a custom command
~~~~~~~~~~~~~~~~~~~~~~~~

Executes a custom command for a given node of a deployment <DeploymentId>.

.. code-block:: bash

     janus deployments custom <DeploymentId> [flags]

Flags:                                                                                                                                                        
  * ``-c``, ``--custom``: Provide the custom command name (use with flag n and i)                                                                       
  * ``-d``, ``--data``: Need to provide the JSON format of the custom command                                                                         
  * ``-i``, ``--inputsMap``: Provide the input for the custom command (use with flag c and n)                                                              
  * ``-n``, ``--node``: Provide the node name (use with flag c and i)           


