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

.. _yorc_plugins_section:

Yorc Plugins (Advanced)
=======================

Yorc exposes several extension points. This section covers the different ways to extend Yorc and how to
create a plugin.

.. note:: This is an advanced section! If you are looking for information on how to use Yorc or on existing plugins
          please refer to our :ref:`main documentation <yorc_index_section>`


Yorc extension points
---------------------

Yorc offers several extension points. The way to provide extensions is to load plugins within Yorc.
Those plugins could be use to enrich Yorc or to override Yorc builtin implementations.
If there is an overlap between a Yorc builtin implementation and an implementation provided by a plugin the plugin implementation
will be preferred.

TOSCA Definitions
~~~~~~~~~~~~~~~~~

One way to extend Yorc is to provide some TOSCA definitions. Those definitions could be used directly within
deployed applications by importing them using the diamond syntax without providing them into the CSAR:

.. code-block:: yaml

  imports:
    - this/is/a/standard/import/to/an/existing/file/within/csar.yaml
    - <my-custom-definition.yaml>
    - <normative-types.yml>

Both two latest imports are taken directly from Yorc not within the CSAR archive.

Delegate Executors
~~~~~~~~~~~~~~~~~~

Some nodes lifecycle could be delegated to Yorc. In this case a workflow does not contain
``set_state`` and ``call_operation`` activities. Their workflow is considered as "delegate"
and acts as a black-box between the initial and started state in the install workflow and
the started to deleted states in the uninstall workflow.

Those kind of executors are typically designed to handle infrastructure types.

You can extend Yorc by adding new implementations that will handle delegate operations for
selected TOSCA types. For those extensions you match a regular expression on the TOSCA type name
to a delegate operation. For instance you can match all delegate operations on type named
``my\.custom\.azure\..*``.

Operation Executors
~~~~~~~~~~~~~~~~~~~

Those kind of executors handle ``call_operation`` activities.

In TOSCA operations are defined by their "implementation artifact". With a plugin you can register
an operation executor for any implementation artifact.

Those executors are typically designed to handle new configuration managers like chef or puppet for
instance.

Infrastructure Usage Collector
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

An infrastructure usage collector allows to retrieve information on underlying infrastructures like
quota usage or cluster load.

You can register a collector for several infrastructures.

How to create a Yorc plugin
---------------------------

Yorc supports a plugin model, plugins are distributed as Go binaries.
Although technically possible to write a plugin in another language, plugin written in Go are the only
implementations officially supported and tested. For more information on installing and configuring Go,
please visit the `Golang installation guide <https://golang.org/doc/install>`_. Yorc and plugins require
at least Go 1.11.

This sections assumes familiarity with Golang and basic programming concepts.

Initial setup and dependencies
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Starting with Yorc 3.2 the official way to handle dependencies is `Go modules <https://github.com/golang/go/wiki/Modules>`_.
This guide will use Go modules to handle dependencies and we recommend to do the same with your plugin.

.. code-block:: bash

  # You can store your code anywhere but if you store it into your GOPATH you need the following line
  $ export GO111MODULE=on
  $ mkdir my-custom-plugin ; cd my-custom-plugin
  $ go mod init github.com/my/custom-plugin
  go: creating new go.mod: module github.com/my/custom-plugin
  $ go get -m github.com/ystia/yorc/v3@v3.2.0-M2
  $ touch main.go


Building the plugin
~~~~~~~~~~~~~~~~~~~

Go requires a main.go file, which is the default executable when the binary is built.
Since Yorc plugins are distributed as Go binaries, it is important to define this
entry-point with the following code:

.. code-block:: Go

  package main

  import (
    "github.com/ystia/yorc/v3/plugin"
  )

  func main() {
    plugin.Serve(&plugin.ServeOpts{})
  }

This establishes the main function to produce a valid, executable Go binary. The contents of
the main function consume Yorc's plugin library. This library deals with all the communication
between Yorc and the plugin.

Next, build the plugin using the Go toolchain:

.. code-block:: bash

  $ go build -o my-custom-plugin

To verify things are working correctly, execute the binary just created:

.. code-block:: bash

  $ ./my-custom-plugin
  This binary is a plugin. These are not meant to be executed directly.
  Please execute the program that consumes these plugins, which will
  load any plugins automatically

Load custom TOSCA definitions
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

You can instruct Yorc to make available some TOSCA definitions as builtin into Yorc.
To do so you need to get the definition content using the way you want. For simplicity we will
use a simple go string variable in the bellow example. Then you need to update ``ServeOpts`` in
your main function.

.. code-block:: Go

  package main

  import (
    "github.com/ystia/yorc/v3/plugin"
  )

  var def = []byte(`tosca_definitions_version: yorc_tosca_simple_yaml_1_0

  metadata:
    template_name: yorc-my-types
    template_author: Yorc
    template_version: 1.0.0

  imports:
    - <normative-types.yml>

  artifact_types:
    mytosca.artifacts.Implementation.MyImplementation:
      derived_from: tosca.artifacts.Implementation
      description: My dummy implementation artifact
      file_ext: [ "myext" ]

  node_types:
    mytosca.types.Compute:
      derived_from: tosca.nodes.Compute
		
  `)

  func main() {
    plugin.Serve(&plugin.ServeOpts{
      Definitions: map[string][]byte{
        "mycustom-types.yml": def,
      },
    })
  }


Implement a delegate executor
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Now we will implement a basic delegate executor, create a file ``delegate.go``
and edit it with following content.

.. code-block:: Go

  package main

  import (
    "context"
    "log"

    "github.com/ystia/yorc/v3/deployments"
    "github.com/ystia/yorc/v3/tasks"
    "github.com/ystia/yorc/v3/tosca"
    "github.com/ystia/yorc/v3/events"
    "github.com/ystia/yorc/v3/config"
  )

  type delegateExecutor struct{}

  func (de *delegateExecutor) ExecDelegate(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
    // Here is how to retrieve config parameters from Yorc config file
    if conf.Infrastructures["my-plugin"] != nil {
      for _, k := range conf.Infrastructures["my-plugin"].Keys() {
        log.Printf("configuration key: %s", k)
      }
      log.Printf("Secret key: %q", conf.Infrastructures["plugin"].GetStringOrDefault("test", "not found!"))
    }

    // Get a consul client to interact with the deployment API
    cc, err := conf.GetConsulClient()
    if err != nil {
      return err
    }
    kv:= cc.KV()

    // Get node instances related to this task (may be a subset of all instances for a scaling operation for instance)
    instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
    if err != nil {
      return err
    }
    
    // Emit events and logs on instance status change 
    for _, instanceName := range instances {
      deployments.SetInstanceStateWithContextualLogs(ctx, kv, deploymentID, nodeName, instanceName, tosca.NodeStateCreating)
    }

    // Use the deployments api to get info about the node to provision
    nodeType, err := deployments.GetNodeType(cc.KV(), deploymentID, nodeName)

    // Emit a log or an event
    events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("Provisioning node %q of type %q", nodeName, nodeType)
    
    for _, instanceName := range instances {
      deployments.SetInstanceStateWithContextualLogs(ctx, kv, deploymentID, nodeName, instanceName, tosca.NodeStateStarted)
    }
    return nil
  }

Now you should instruct the plugin system that a new executor is available and which types it supports.
This could be done by altering again  ``ServeOpts`` in your main function.


.. code-block:: Go

  package main

  import (
    "github.com/ystia/yorc/v3/plugin"
    "github.com/ystia/yorc/v3/prov"
  )

  // ... omitted for brevity ...

  func main() {
    plugin.Serve(&plugin.ServeOpts{
      Definitions: map[string][]byte{
        "mycustom-types.yml": def,
      },
      DelegateSupportedTypes: []string{`mytosca\.types\..*`},
      DelegateFunc: func() prov.DelegateExecutor {
        return new(delegateExecutor)
      },
    })
  }

Implement an operation executor
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

An operation executor could be implemented exactly in the same way than a delegate executor,
expect that it need to support two different functions, ``ExecOperation`` and ``ExecOperationAsync``.
The first one is the more common usecase while the latest is designed to handle asynchronous
(non-blocking for long running) operations, like jobs execution typically.
In this guide we will focus on ``ExecOperation`` please read our documentation about jobs for more
details on asynchronous operations.
You can create a ``operation.go`` file with following content.

.. code-block:: Go

  package main

  import (
    "context"
    "fmt"
    "time"

    "github.com/ystia/yorc/v3/config"
    "github.com/ystia/yorc/v3/events"
    "github.com/ystia/yorc/v3/prov"
  )

  type operationExecutor struct{}

  func (oe *operationExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {
    return nil, 0, fmt.Errorf("asynchronous operations %v not yet supported by this sample", operation)
  }

  func (oe *operationExecutor) ExecOperation(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
    events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("Hello from my OperationExecutor")
    // Your business logic goes there
    return nil
  }

Then you should instruct the plugin system that a new executor is available and which implementation artifacts it supports.
Again, this could be done by altering ``ServeOpts`` in your main function.


.. code-block:: Go

  // ... omitted for brevity ...

  func main() {
    plugin.Serve(&plugin.ServeOpts{
      Definitions: map[string][]byte{
        "mycustom-types.yml": def,
      },
      DelegateSupportedTypes: []string{`mytosca\.types\..*`},
      DelegateFunc: func() prov.DelegateExecutor {
        return new(delegateExecutor)
      },
      OperationSupportedArtifactTypes: []string{"mytosca.artifacts.Implementation.MyImplementation"},
      OperationFunc: func() prov.OperationExecutor {
        return new(operationExecutor)
      },
    })
  }


Using Your Plugin
~~~~~~~~~~~~~~~~~

First your plugin should be dropped into Yorc's plugins directory before starting Yorc. Yorc's :ref:`plugins
directory is configurable <option_terraform_plugins_dir_cmd>` but by default it's a directory named ``plugins`` in
the current directory when Yorc is launched.

By exporting an environment variable ``YORC_LOG=1`` before running Yorc, debug logs will be displayed.

.. code-block:: Bash

  # Run consul in a terminal
  $ consul agent -dev
  # Run Yorc in another terminal
  $ mkdir plugins
  $ cp my-custom-plugin plugins/
  $ YORC_LOG=1 yorc server
  ...
  2019/02/12 14:28:23 [DEBUG] Loading plugin "/tmp/yorc/plugins/my-custom-plugin"...
  2019/02/12 14:28:23 [INFO]  30 workers started
  2019/02/12 14:28:23 [DEBUG] plugin: starting plugin: /tmp/yorc/plugins/my-custom-plugin []string{"/tmp/yorc/plugins/my-custom-plugin"}
  2019/02/12 14:28:23 [DEBUG] plugin: waiting for RPC address for: /tmp/yorc/plugins/my-custom-plugin
  2019/02/12 14:28:23 [DEBUG] plugin: my-custom-plugin: 2019/02/12 14:28:23 [DEBUG] plugin: plugin address: unix /tmp/plugin262069315
  2019/02/12 14:28:23 [DEBUG] plugin: my-custom-plugin: 2019/02/12 14:28:23 [DEBUG] Consul Publisher created with a maximum of 500 parallel routines.
  2019/02/12 14:28:23 [DEBUG] Registering supported node types [mytosca\.types\..*] into registry for plugin "my-custom-plugin"
  2019/02/12 14:28:23 [DEBUG] Registering supported implementation artifact types [mytosca.artifacts.Implementation.MyImplementation] into registry for plugin "my-custom-plugin"
  2019/02/12 14:28:23 [DEBUG] Registering TOSCA definition "mycustom-types.yml" into registry for plugin "my-custom-plugin"
  2019/02/12 14:28:23 [INFO]  Plugin "my-custom-plugin" successfully loaded
  2019/02/12 14:28:23 [INFO]  Starting HTTPServer on address [::]:8800
  ...

Now you can create a dummy TOSCA application ``topology.yaml``

.. code-block:: yaml

  tosca_definitions_version: alien_dsl_2_0_0

  metadata:
    template_name: TestPlugins
    template_version: 0.1.0-SNAPSHOT
    template_author: admin

  imports:
    - <mycustom-types.yml>

  node_types:
    my.types.Soft:
      derived_from: tosca.nodes.SoftwareComponent
      interfaces:
        Standard:
          create: dothis.myext

  topology_template:
    node_templates:
      Compute:
        type: mytosca.types.Compute
        capabilities:
          endpoint:
            properties:
              protocol: tcp
              initiator: source
              secure: true
              network_name: PRIVATE
          scalable:
            properties:
              max_instances: 5
              min_instances: 1
              default_instances: 2

      Soft:
        type: my.types.Soft

    workflows:
      install:
        steps:
          Compute_install:
            target: Compute
            activities:
              - delegate: install
            on_success:
              - Soft_creating
          Soft_creating:
            target: Soft
            activities:
              - set_state: creating
            on_success:
              - create_Soft
          create_Soft:
            target: Soft
            activities:
              - call_operation: Standard.create
            on_success:
              - Soft_created
          Soft_created:
            target: Soft
            activities:
              - set_state: created
            on_success:
              - Soft_started
          Soft_started:
            target: Soft
            activities:
              - set_state: started
      uninstall:
        steps:
          Soft_deleted:
            target: Soft
            activities:
              - set_state: deleted
            on_success:
              - Compute_uninstall
          Compute_uninstall:
            target: Compute
            activities:
              - delegate: uninstall

Finally you can deploy your application and see (among others) the following logs:

.. code-block:: bash

  $ yorc d deploy -l --id my-app topology.yaml
  <...>
  [2019-02-12T16:51:55.207420877+01:00][INFO][my-app][install][5a7638e8-dde2-48e7-9e5a-89350ccd99a7][8f4b31da-8f27-456e-8c25-0520366bda30-0][Compute][0][delegate][install][]Status for node "Compute", instance "0" changed to "creating"
  [2019-02-12T16:51:55.20966624+01:00][INFO][my-app][install][5a7638e8-dde2-48e7-9e5a-89350ccd99a7][8f4b31da-8f27-456e-8c25-0520366bda30-1][Compute][1][delegate][install][]Status for node "Compute", instance "1" changed to "creating"
  [2019-02-12T16:51:55.211403476+01:00][INFO][my-app][install][5a7638e8-dde2-48e7-9e5a-89350ccd99a7][8f4b31da-8f27-456e-8c25-0520366bda30][Compute][][delegate][install][]Provisioning node "Compute" of type "mytosca.types.Compute"
  [2019-02-12T16:51:55.213793985+01:00][INFO][my-app][install][5a7638e8-dde2-48e7-9e5a-89350ccd99a7][8f4b31da-8f27-456e-8c25-0520366bda30-0][Compute][0][delegate][install][]Status for node "Compute", instance "0" changed to "started"
  [2019-02-12T16:51:55.215991445+01:00][INFO][my-app][install][5a7638e8-dde2-48e7-9e5a-89350ccd99a7][8f4b31da-8f27-456e-8c25-0520366bda30-1][Compute][1][delegate][install][]Status for node "Compute", instance "1" changed to "started"
  <...>
  [2019-02-12T16:51:55.384726783+01:00][INFO][my-app][install][5a7638e8-dde2-48e7-9e5a-89350ccd99a7][3d640a5a-3093-4c7b-83e4-c67e57a1c430][Soft][][standard][create][]Hello from my OperationExecutor
  <...>
  [2019-02-12T16:51:55.561607771+01:00][INFO][my-app][install][5a7638e8-dde2-48e7-9e5a-89350ccd99a7][][][][][][]Status for deployment "my-app" changed to "deployed"


Et voilà !
