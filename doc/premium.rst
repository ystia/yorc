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

.. _yorc_premium_section:

Premium features
================

The following features are only available with Yorc premium. Please, contact us for more information on Premium offers.

Providing scripts outputs live in Yorc logs
-------------------------------------------

The open source version of Yorc now provides ansible playbooks logs for each task once the task is finished. Scripts outputs are still provided in Yorc logs at the end of the script execution.

The premium version of Yorc will provide scripts stdout and stderr live in Yorc logs, so that scripts having a long execution time can be monitored from Yorc through their output.

Deployment update
-----------------

It's possible to update a deployed topology by making some actions in the application.

Add/Remove/Update workflows
~~~~~~~~~~~~~~~~~~~~~~~~~~~

This feature allows to add new workflows, remove or modify existing ones.

Add / remove monitoring policies
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

HTTP and TCP monitoring policies can be applied on an application in order to monitor Software components or Compute instances liveness.
See `Applying policies <https://yorc-a4c-plugin.readthedocs.io/en/latest/policies.html>`_ for more information.

With the Premium version, you can add new monitoring policies on a deployed application if you miss it when you deploy the app.
You can also modify or remove existing monitoring policies on a deployed application if your needs changed.

Update TOSCA Types
~~~~~~~~~~~~~~~~~~

This feature allows to update imported Tosca types either in the same version or in a new version in order to support new attributes, properties or even operations.
Mixed with a new custom workflow, this allows to execute by instance new operations.

Add / remove nodes
~~~~~~~~~~~~~~~~~~

This feature allows to add new node templates in a deployed topology or to remove existing ones.
In the first implementation, it's not possible to mix adds and removes in the same update, you need to do it in different updates.