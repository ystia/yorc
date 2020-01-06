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

.. _yorc_telemetry_section:

Yorc Telemetry
===============

Yorc collects various runtime metrics. These metrics are aggregated on a ten second interval and are retained for one minute.

To view this data, you must send a signal to the Yorc process: on Unix, this is USR1 while on Windows it is BREAK. Once Yorc receives the signal, it will dump the current telemetry information to stderr.

Telemetry information can be streamed to both statsite as well as statsd or pull from Prometheus based on providing the appropriate configuration options. See :ref:`yorc_config_file_telemetry_section` for more information.

Below is sample output (lot of metrics omitted for brevity) of a telemetry dump::

    [2020-01-06 16:31:00 +0200 CEST][G] 'yorc.runtime.alloc_bytes': 73723728.000
    [2020-01-06 16:31:00 +0200 CEST][G] 'yorc.workers.free': 2.000
    [2020-01-06 16:31:00 +0200 CEST][C] 'yorc.http.GET.metrics.200': Count: 2 Sum: 2.000 LastUpdated: 2020-01-06 16:31:06.253380804 +0200 CEST
    [2020-01-06 16:31:00 +0200 CEST][S] 'yorc.tasks.maxBlockTimeMs': Count: 10 Sum: 0.000 LastUpdated: 2020-01-06 16:31:09.805073861 +0200 CEST
    [2020-01-06 16:31:00 +0200 CEST][S] 'yorc.http.duration.GET.events': Count: 2 Min: 27.765 Mean: 29.474 Max: 31.183 Stddev: 2.417 Sum: 58.948 LastUpdated: 2020-01-06 16:31:06.253392224 +0200 CEST
    [2020-01-06 16:31:00 +0200 CEST][S] 'yorc.http.duration.GET.server/health': Count: 10 Min: 27.765 Mean: 21.274 Max: 32.193 Stddev: 1.527 Sum: 65.848 LastUpdated: 2020-01-06 16:31:06.2557253638 +0200 CEST
    [2020-01-06 16:31:10 +0200 CEST][S] 'yorc.tasks.maxBlockTimeMs': Count: 10 Sum: 0.000 LastUpdated: 2020-01-06 16:31:19.986227315 +0200 CEST

Another exemple in which the telemetry service configuration property ``disable_hostname`` is set to **false** and the host name is ``yorc-server-0``:

    [2020-01-06 16:31:00 +0200 CEST][G] 'yorc.yorc-server-0.runtime.alloc_bytes': 73723728.000
    [2020-01-06 16:31:00 +0200 CEST][G] 'yorc.yorc-server-0.workers.free': 2.000

Key metrics
-----------

Metric Types
~~~~~~~~~~~~

+---------+---------------------------------------------------------------------------------------------------------------------+-----------+
|  Type   |                                                     Description                                                     | Quantiles |
+=========+=====================================================================================================================+===========+
| Gauge   | Gauge types report an absolute number at the end of the aggregation interval.                                       | false     |
+---------+---------------------------------------------------------------------------------------------------------------------+-----------+
| Counter | Counts are incremented and flushed at the end of the aggregation interval and then are reset to zero.               | true      |
+---------+---------------------------------------------------------------------------------------------------------------------+-----------+
| Timer   | Timers measure the time to complete a task and will include quantiles, means, standard deviation, etc per interval. | true      |
+---------+---------------------------------------------------------------------------------------------------------------------+-----------+


Go Runtime metrics
~~~~~~~~~~~~~~~~~~
.. 
   MAG - According to:
   https://github.com/sphinx-doc/sphinx/issues/3043
   http://www.sphinx-doc.org/en/stable/markup/misc.html#tables
.. tabularcolumns:: |p{0.20\textwidth}|p{0.55\textwidth}|p{0.10\textwidth}|p{0.05\textwidth}|

+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
|            Metric Name             |                                           Description                                            |       Unit        | Metric Type |
|                                    |                                                                                                  |                   |             |
+====================================+==================================================================================================+===================+=============+
| ``yorc.runtime.num_goroutines``    | This tracks the number of running goroutines and is a general load pressure                      | number            | gauge       |
|                                    | indicator. This may burst from time to time but should return to a steady                        | of                |             |
|                                    | state value.                                                                                     | goroutines        |             |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.alloc_bytes``       | This measures the number of bytes allocated by the Yorc process. This may                        | bytes             | gauge       |
|                                    | burst from time to time but should return to a steady state value.                               |                   |             |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.heap_objects``      | This measures the number of objects allocated on the heap and is a general memory                |                   |             |
|                                    | pressure indicator. This may burst from time to time but should return to a steady state value.  | number of objects | gauge       |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.sys``               | Sys is the total bytes of memory obtained from the OS.Sys measures the virtual address space     |                   |             |
|                                    | reserved by the Go runtime for the  heap, stacks, and other                                      | bytes             | gauge       |
|                                    | internal data structures. It's likely that not all of the virtual address space is backed        |                   |             |
|                                    | by physical memory at any given moment, though in general it all was at some point.              |                   |             |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.malloc_count``      | Mallocs is the cumulative count of heap objects allocated. The number of live objects is         | number of Mallocs | gauge       |
|                                    | Mallocs - Frees.                                                                                 |                   |             |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.free_count``        | Frees is the cumulative count of heap objects freed.                                             | number of frees   | gauge       |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.total_gc_pause_ns`` | PauseTotalNs is the cumulative nanoseconds in GC stop-the-world pauses since the program         | nanoseconds       | gauge       |
|                                    | started.                                                                                         |                   |             |
|                                    | During a stop-the-world pause, all goroutines are paused and only the garbage collector can run. |                   |             |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.total_gc_runs``     | Gc runs is the number of completed GC cycles.                                                    | number of cycles  | gauge       |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+
| ``yorc.runtime.gc_pause_ns``       | Latest GC run stop-the-world pause duration.                                                     | nanoseconds       | timer       |
+------------------------------------+--------------------------------------------------------------------------------------------------+-------------------+-------------+

Yorc REST API metrics
~~~~~~~~~~~~~~~~~~~~~

The **method** lablel represents the HTTP verb.
The **path** lablel corresponds to the request URL where slashes are replaced by dashes.
The **status** label represents a HTTP status codes (ie: 200, 404, 500, ...).

+------------------------+--------------------+------------------------------------------------------+--------------------+-------------+
|       Metric Name      |        Labels      |              Description                             |        Unit        | Metric Type |
|                        |                    |                                                      |                    |             |
+========================+====================+======================================================+====================+=============+
| ``yorc.http.duration`` | method             | This measures the duration of an API call.           | milliseconds       | timer       |
|                        | path               |                                                      |                    |             |
+------------------------+--------------------+------------------------------------------------------+--------------------+-------------+
| ``yorc.http.total``    | method             |  This counts the number of API calls                 | number of requests | counter     |
|                        | path               |                                                      |                    |             |
|                        | status             |                                                      |                    |             |
+------------------------+--------------------+------------------------------------------------------+--------------------+-------------+

Yorc Workers metrics
~~~~~~~~~~~~~~~~~~~~

+---------------------------------------+-------------------------------------------------------------------+------------------------+-------------+
|              Metric Name              |                               Description                         |      Unit              | Metric Type |
+=======================================+===================================================================+========================+=============+
| ``yorc.workers.free``                 | This tracks the number of free Yorc workers.                      | number of free workers | gauge       |
+---------------------------------------+-------------------------------------------------------------------+------------------------+-------------+

Yorc Tasks metrics
~~~~~~~~~~~~~~~~~~

+----------------------------------+--------------------------------------------------------------+-----------------+-------------+
|           Metric Name            |                        Description                           |      Unit       | Metric Type |
|                                  |                                                              |                 |             |
+==================================+==============================================================+=================+=============+
| ``yorc.tasks.maxBlockTimeMs``    |  Measures the highest duration since creation                | milliseconds    | timer       |
|                                  |          for all waiting tasks.                              |                 |             |
+----------------------------------+--------------------------------------------------------------+-----------------+-------------+
| ``yorc.tasks.nbWaiting``         |  Tracks the number of tasks waiting for being processed      | number of tasks | gauge       |
+----------------------------------+--------------------------------------------------------------+-----------------+-------------+
| ``yorc.tasks.wait``              |  Measures the finally waited time for a task being processed | milliseconds    | timer       |
+----------------------------------+--------------------------------------------------------------+-----------------+-------------+

Yorc taskExecutions metrics
~~~~~~~~~~~~~~~~~~~~~~~~~~~
The **Deployment** label is set to the deployment ID of the monitored taskExecution.
The **Step** label represents the name of the step that is execuded.
The **Type** label corresponds to the task type (``Deploy``, ``Undeploy``, ``Purge``, ``ScaleOut``, ``CustomCommand``, etc.)


+---------------------------------------+-----------------------------+-------------------------------------------------+-----------------+-------------+
|           Metric Name                 |         Labels              |                Description                      |      Unit       | Metric Type |
|                                       |                             |                                                 |                 |             |
+=======================================+=============================+=================================================+=================+=============+
| ``yorc.taskExecutions.<FinalStatus>`` | Deployment                  | Counts the number of taskExecutions ending      | number of tasks | counter     |
|                                       | Type                        |         in a given final status                 |                 |             |
+---------------------------------------+-----------------------------+-------------------------------------------------+-----------------+-------------+
| ``yorc.taskExecutions.duration``      | Deployment                  | Measures a taskExecution's processing duration  | milliseconds    | timer       |
|                                       | Type                        |                                                 |                 |             |
+---------------------------------------+-----------------------------+-------------------------------------------------+-----------------+-------------+

Yorc Executors metrics
~~~~~~~~~~~~~~~~~~~~~~

There are two types of executors in Yorc: ``delegate`` executors and ``operation`` executors. 
Delegate executors handle the deployment of Yorc natively supported TOSCA nodes (like an OpenStack compute for instance).
Operation executors handle implementations of an lifecycle operation provided as part of the TOSCA node definition (like a shell script or an ansible playbook).

In the below table <ExecType> is the executor type (``operation`` or ``delegate``).


+---------------------------------------+-------------------+------------------------------------------------+---------------------+-------------+
|           Metric Name                 |     Labels        |                Description                     |      Unit           | Metric Type |
|                                       |                   |                                                |                     |             |
+=======================================+===================+================================================+=====================+=============+
| ``yorc.executor.<ExecType>``          | Deployment        | This measures the duration of an execution.    | milliseconds        | timer       |
|                                       | Node              |                                                |                     |             |
|                                       | Name              |                                                |                     |             |
+---------------------------------------+-------------------+------------------------------------------------+---------------------+-------------+
| ``yorc.executor.<ExecType>.failures`` | Deployment        | Counts the number of failed executions.        | number of failures  | counter     |
|                                       | Node              |                                                |                     |             |
|                                       | Name              |                                                |                     |             |
+---------------------------------------+-------------------+------------------------------------------------+---------------------+-------------+
| ``yorc.executor.<ExecType>.successes``| Deployment        | Counts the number of successful executions.    | number of successes | counter     |
|                                       | Node              |                                                |                     |             |
|                                       | Name              |                                                |                     |             |
+---------------------------------------+-------------------+------------------------------------------------+---------------------+-------------+

The **Deployment** label is set to the deployment ID, and the **Node** label is set to the fully qualified TOSCA node type where dots were replaced by
dashes.
The **Name** label is set to the TOSCA operation name where dots were replaced by dashes.

Yorc Actions scheduling metrics
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~


+----------------------------------------+-----------------------+------------------------------------------------+---------------------+-------------+
|           Metric Name                  |         Labels        |                Description                     |      Unit           | Metric Type |
|                                        |                       |                                                |                     |             |
+========================================+=======================+================================================+=====================+=============+
| ``yorc.scheduling.ticks``              |  ActionType           | Counts the number of action schedulings.       | number of schedules | counter     |
|                                        |  ActionID             |                                                |                     |             |
+----------------------------------------+-----------------------+------------------------------------------------+---------------------+-------------+
| ``yorc.scheduling.misses``             | ActionType            | Counts the number of missed trigger due to     | number of missed    | counter     |
|                                        | ActionID              | another execution already planned or running   | schedules           |             |
|                                        | TaskID                |                                                |                     |             |
+----------------------------------------+-----------------------+------------------------------------------------+---------------------+-------------+

If an action schedule misses because another task is already executing it, the TaskID label contains this task's ID.

Yorc SSH connection pool
~~~~~~~~~~~~~~~~~~~~~~~~

+---------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
|                            Metric Name            |                             Description                              |         Unit         | Metric Type |
|                                                   |                                                                      |                      |             |
+===================================================+======================================================================+======================+=============+
| ``yorc.ssh-connections-pool.creations``           | This measures the number of created connections.                     | number of connection | counter     |
+---------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.create-failed``       | This measures the number of failed create connections.               | number of connection | counter     |
|                                                   |                                                                      | create failures      |             |
+---------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.closes``              | This measures the number of closed connections.                      | number of close      | counter     |
+---------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+


Measures about the utilisation of sessions related to ssh connections. 

+---------------------------------------------------+----------------+----------------------------------------------------------------------+----------------------+-------------+
|                            Metric Name            |    Labels      |                             Description                              |         Unit         | Metric Type |
|                                                   |                |                                                                      |                      |             |
+===================================================+================+======================================================================+======================+=============+
| ``yorc.ssh-connections-pool.sessions.creations``  | ConnectionName | This measures the number of sessions created for a given connection. | number of open       | counter     |
+---------------------------------------------------+----------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.sessions.closes``     | ConnectionName | This measures the number of sessions closed for a given connection.  | number of close      | counter     |
+---------------------------------------------------+----------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.sessions.open-failed``| ConnectionName | This tracks the number of failures when opening an SSH session       | number of open       | counter     |
|                                                   |                | (multiplexed on top of an existing connection).                      | failures             |             |
+---------------------------------------------------+----------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.sessions.open``       | ConnectionName | This tracks the number of currently open sessions per connection     | number of sessions   | gauge       |
|                                                   |                |                                                                      |                      |             |
+---------------------------------------------------+----------------+----------------------------------------------------------------------+----------------------+-------------+