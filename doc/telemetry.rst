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

    [2017-07-19 16:31:00 +0200 CEST][G] 'yorc.yorc-server-0.runtime.alloc_bytes': 73723728.000
    [2017-07-19 16:31:00 +0200 CEST][G] 'yorc.yorc-server-0.workers.free': 2.000
    [2017-07-19 16:31:00 +0200 CEST][C] 'yorc.http.200.GET.metrics': Count: 2 Sum: 2.000 LastUpdated: 2017-07-19 16:31:06.253380804 +0200 CEST
    [2017-07-19 16:31:00 +0200 CEST][S] 'yorc.tasks.maxBlockTimeMs': Count: 10 Sum: 0.000 LastUpdated: 2017-07-19 16:31:09.805073861 +0200 CEST
    [2017-07-19 16:31:00 +0200 CEST][S] 'yorc.http.GET.metrics': Count: 2 Min: 27.765 Mean: 29.474 Max: 31.183 Stddev: 2.417 Sum: 58.948 LastUpdated: 2017-07-19 16:31:06.253392224 +0200 CEST
    [2017-07-19 16:31:10 +0200 CEST][S] 'yorc.tasks.maxBlockTimeMs': Count: 10 Sum: 0.000 LastUpdated: 2017-07-19 16:31:19.986227315 +0200 CEST
    [2017-07-19 16:31:20 +0200 CEST][C] 'yorc.http.200.GET.metrics': Count: 2 Sum: 2.000 LastUpdated: 2017-07-19 16:31:26.257243322 +0200 CEST
    [2017-07-19 16:31:20 +0200 CEST][S] 'yorc.tasks.maxBlockTimeMs': Count: 9 Sum: 0.000 LastUpdated: 2017-07-19 16:31:29.138694946 +0200 CEST
    [2017-07-19 16:31:20 +0200 CEST][S] 'yorc.http.GET.metrics': Count: 2 Min: 32.371 Mean: 41.727 Max: 51.083 Stddev: 13.232 Sum: 83.454 LastUpdated: 2017-07-19 16:31:26.257253638 +0200 CEST


Key metrics
-----------

Metric Types
~~~~~~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~~~~~~~

+----------------------------------------+-------------------------------------------------------------------------------------+--------------------+-------------+
|              Metric Name               |                                     Description                                     |        Unit        | Metric Type |
|                                        |                                                                                     |                    |             |
+========================================+=====================================================================================+====================+=============+
| ``yorc.http.<Method>.<Path>``          | This measures the duration of an API call. <Method> is the HTTP verb and <Path> the | milliseconds       | timer       |
|                                        | Path part of the URL where slashes are replaced by dashes.                          |                    |             |
+----------------------------------------+-------------------------------------------------------------------------------------+--------------------+-------------+
| ``yorc.http.<Status>.<Method>.<Path>`` | This counts the number of API calls by HTTP status codes (ie: 200, 404, 500, ...)   | number of requests | counter     |
|                                        | , HTTP verb and URL path as described above.                                        |                    |             |
+----------------------------------------+-------------------------------------------------------------------------------------+--------------------+-------------+

Yorc Workers & Tasks metrics
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

+---------------------------------------+--------------------------------------------------------------------------+-----------------+-------------+
|              Metric Name              |                               Description                                |      Unit       | Metric Type |
|                                       |                                                                          |                 |             |
+=======================================+==========================================================================+=================+=============+
| ``yorc.workers.free``                 | This tracks the number of free Yorc workers.                             | number of free  | gauge       |
|                                       |                                                                          | workers         |             |
+---------------------------------------+--------------------------------------------------------------------------+-----------------+-------------+
| ``tasks.maxBlockTimeMs``              | This measures the highest duration since creation for all waiting tasks. | milliseconds    | timer       |
+---------------------------------------+--------------------------------------------------------------------------+-----------------+-------------+
| ``tasks.nbWaiting``                   | This tracks the number of tasks waiting for being processed.             | number of       | gauge       |
|                                       |                                                                          | tasks           |             |
+---------------------------------------+--------------------------------------------------------------------------+-----------------+-------------+
| ``tasks.wait``                        | This measures the finally waited time for a task being processed.        | milliseconds    | timer       |
+---------------------------------------+--------------------------------------------------------------------------+-----------------+-------------+
| ``task.<DepID>.<Type>.<FinalStatus>`` | This counts by deployment and task type the final status of a task.      | number of tasks | counter     |
+---------------------------------------+--------------------------------------------------------------------------+-----------------+-------------+
| ``task.<DepID>.<Type>``               | This measures the task processing duration.                              | milliseconds    | timer       |
+---------------------------------------+--------------------------------------------------------------------------+-----------------+-------------+


Yorc Executors metrics
~~~~~~~~~~~~~~~~~~~~~~~

There are two types of executors in Yorc "delegates executors" and "operations executors". Delegates executors handle the deployment of Yorc natively supported
TOSCA nodes (like an Openstack compute for instance) while Operations executors handle implementations of an lifecycle operations provided as part of the TOSCA node
definition (like a shell script or an ansible playbook).

In the below table <ExecType> is the executor type, <DepID> the deployment ID, <NodeType> the fully qualified TOSCA node type where dots where replaced by
dashes and <OpName> the TOSCA operation name where dots where replaced by dashes.

+--------------------------------------------------------------------+--------------------------------------------------+---------------------+-------------+
|                            Metric Name                             |                   Description                    |        Unit         | Metric Type |
|                                                                    |                                                  |                     |             |
+====================================================================+==================================================+=====================+=============+
| ``yorc.executor.<ExecType>.<DepID>.<NodeType>.<OpName>``           | This measures the duration of an execution.      | milliseconds        | timer       |
+--------------------------------------------------------------------+--------------------------------------------------+---------------------+-------------+
| ``yorc.executor.<ExecType>.<DepID>.<NodeType>.<OpName>.failures``  | This counts the number of failed executions.     | number of failures  | counter     |
+--------------------------------------------------------------------+--------------------------------------------------+---------------------+-------------+
| ``yorc.executor.<ExecType>.<DepID>.<NodeType>.<OpName>.successes`` | This counts the number of successful executions. | number of successes | counter     |
+--------------------------------------------------------------------+--------------------------------------------------+---------------------+-------------+

Yorc SSH connection pool
~~~~~~~~~~~~~~~~~~~~~~~~

+--------------------------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
|                            Metric Name                             |                             Description                              |         Unit         | Metric Type |
|                                                                    |                                                                      |                      |             |
+====================================================================+======================================================================+======================+=============+
| ``yorc.ssh-connections-pool.<connection_id>.sessions.open-failed`` | This tracks the number of failures when opening an SSH session       | number of            | counter     |
|                                                                    | (multiplexed on top of an existing connection).                      | failures             |             |
+--------------------------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.<connection_id>.sessions.creations``   | This measures the number of sessions created for a given connection. | number of sessions   | counter     |
+--------------------------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.<connection_id>.sessions.closes``      | This measures the number of sessions closed for a given connection.  | number of sessions   | counter     |
+--------------------------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.<connection_id>.sessions.open``        | This tracks the number of currently open sessions per connection     | number of sessions   | gauge       |
|                                                                    |                                                                      |                      |             |
+--------------------------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.creations.<connection_id>``            | This measures the number of created connections.                     | number of connection | counter     |
+--------------------------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
| ``yorc.ssh-connections-pool.closes.<connection_id>``               | This measures the number of closed connections.                      | number of connection | counter     |
+--------------------------------------------------------------------+----------------------------------------------------------------------+----------------------+-------------+
