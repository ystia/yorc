# Yorc HTTP (REST) API

yorc runs an HTTP server that exposes an API in a restful manner.
Currently supported urls are:

## Deployments

Adding the 'pretty' url parameter to your requests allow to generate an indented json output.

### Submit a CSAR to deploy <a name="submit-csar"></a>

Creates a new deployment by uploading a CSAR. 'Content-Type' header should be set to 'application/zip'.

There are two ways to submit a new deployment, you can let yorc generate a unique deployment ID or you can specify it.

#### Auto-generated deployment ID

In this case you should use a `POST` method.

`POST /deployments`

#### Customized deployment ID

In this case you should use a `PUT` method. There are some constraints on submitting a deployment with a given ID:

* This ID should respect the following format: `^[-_0-9a-zA-Z]+$` and be less than 36 characters long (otherwise a `400 BadRequest` error is returned)
* This ID should not already be in used (otherwise a `409 Conflict` error is returned)

`PUT /deployments/<deployment_id>`

**Result**:

In both submission ways, a successfully submitted deployment will result in an HTTP status code 201 with a 'Location' header relative to the base URI indicating the task URI handling the deployment process.

```HTTP
HTTP/1.1 201 Created
Location: /deployments/b5aed048-c6d5-4a41-b7ff-1dbdc62c03b0/tasks/b4144668-5ec8-41c0-8215-842661520147
Content-Length: 0
```

This endpoint produces no content except in case of error.

A critical note is that the deployment is proceeded asynchronously and a success only guarantees that the deployment is successfully
**submitted**.

### Update a deployment (premium feature) <a name="update-csar"></a>

Updates a deployment by uploading an updated CSAR. 'Content-Type' header should be set to 'application/zip'.

`PATCH /deployments/<deployment_id>`

**Result**:

A successfully submitted deployment update will result in an HTTP status code 200.
There won't be any 'Location' header relative to the base URI indicating a task URI
handling the update process, as the current implementation only supports the update
of Workflows, which is performed synchronously.

```HTTP
HTTP/1.1 200 OK
Content-Length: 0
```

This endpoint produces no content except in case of error.
As the ability to update a deployment is a premium feature, attempting to perform
a deployment update using the open source version of Yorc will return the error `403 Forbidden`.

### List deployments <a name="list-deps"></a>

Retrieves the list of deployments. 'Accept' header should be set to 'application/json'.

`GET /deployments`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "deployments": [
    {
      "id": "deployment1",
      "status": "DEPLOYED",
      "links": [
        {
          "rel": "deployment",
          "href": "/deployments/deployment1",
          "type": "application/json"
        }
      ]
    }
  ]
}
```

### Undeploy  an active deployment <a name="undeploy"></a>

Undeploy a deployment. By adding the optional 'purge' url parameter to your request you will suppress any reference to this deployment from the yorc database at the end of the undeployment. A successful call to this endpoint results in a HTTP status code 202 with a 'Location' header relative to the base URI indicating the task URI handling the undeployment process.
By adding the optional 'stopOnError' url parameter to your request, the un-deployment will stop at the first encountered error. Otherwise, it will continue until the end.

`DELETE /deployments/<deployment_id>[?purge]&[stopOnError]`

**Response**:

```HTTP
HTTP/1.1 202 Accepted
Location: /deployments/b5aed048-c6d5-4a41-b7ff-1dbdc62c03b0/tasks/b4144668-5ec8-41c0-8215-842661520147
Content-Length: 0
```

This endpoint produces no content except in case of error.

A critical note is that the undeployment is proceeded asynchronously and a success only guarantees that the undeployment task is successfully
**submitted**.

### Purge a deployment <a name="purge"></a>

Purge a deployment. This deployment should be in UNDEPLOYED status.
If an error is encountered the purge process is stopped and the deployment status is set
to PURGE_FAILED.

A purge may be run in force mode. In this mode Yorc does not check if the deployment is in
UNDEPLOYED status or even if the deployment exist. Moreover, in force mode the purge process
doesn't fail-fast and try to delete as much as it can. An report with encountered errors is
produced at the end of the process.

`POST /deployments/<deployment_id>/purge[?force]`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Length: 21
Content-Type: application/json
```

```json
{
  "errors": null
}
```

`errors` field is a list as force mode may encounter several errors below is an example:

```json
{
  "errors": [
    {
      "id": "internal_server_error",
      "status": 500,
      "title": "Internal Server Error",
      "detail": "failed to remove deployments artifacts stored on disk: \"work/deployments/TestSRun-Environment\": unlinkat work/deployments/TestSRun-Environment/dep: permission denied"
    },
    {
      "id": "internal_server_error",
      "status": 500,
      "title": "Internal Server Error",
      "detail": "Something went wrong: unlinkat work/store/_yorc/deployments/TestSRun-Environment/dep: permission denied"
    }
  ]
}
```

**Note:**

* Unlike the [Undeploy API endpoint](#undeploy) with purge option, this endpoint is synchronous.
* This endpoint is transitory to ensure backward compatibility on the 4.x release, see [issue #710](https://github.com/ystia/yorc/issues/710) for more details.

### Get the deployment information <a name="dep-info"></a>

Retrieve the deployment status and the list (as Atom links) of the nodes and tasks related the deployment.

'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "id": "55d54226-5ce5-4278-96e4-97dd4cbb4e62",
  "status": "DEPLOYED",
  "links": [
    {
      "rel": "self",
      "href": "/deployments/55d54226-5ce5-4278-96e4-97dd4cbb4e62",
      "type": "application/json"
    },
    {
      "rel": "node",
      "href": "/deployments/6ce5419f-2ce5-44d5-ac51-fbe425bd59a2/nodes/Apache",
      "type": "application/json"
    },
    {
      "rel": "node",
      "href": "/deployments/6ce5419f-2ce5-44d5-ac51-fbe425bd59a2/nodes/ComputeRegistry",
      "type": "application/json"
    },
    {
      "rel": "node",
      "href": "/deployments/6ce5419f-2ce5-44d5-ac51-fbe425bd59a2/nodes/PHP",
      "type": "application/json"
    },
    {
      "rel": "task",
      "href": "/deployments/6ce5419f-2ce5-44d5-ac51-fbe425bd59a2/tasks/b4144668-5ec8-41c0-8215-842661520147",
      "type": "application/json"
    }

  ]
}
```

### Get the deployment information about a given node <a name="node-info"></a>

Retrieve the node status and the list (as Atom links) of the instances for this node.

'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/nodes/<node_name>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "name": "ComputeB",
  "status": "started",
  "links": [
    {
      "rel": "self",
      "href": "/deployments/6ce5419f-2ce5-44d5-ac51-fbe425bd59a2/nodes/ComputeB",
      "type": "application/json"
    },
    {
      "rel": "instance",
      "href": "/deployments/6ce5419f-2ce5-44d5-ac51-fbe425bd59a2/nodes/ComputeB/instances/0",
      "type": "application/json"
    },
    {
      "rel": "instance",
      "href": "/deployments/6ce5419f-2ce5-44d5-ac51-fbe425bd59a2/nodes/ComputeB/instances/1",
      "type": "application/json"
    }
  ]
}
```

### Get the deployment information about a given node instance <a name="instance-info"></a>

Retrieve a node instance's status.
Get the list (as Atom links) of the attributes for this instance.

'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/nodes/<node_name>/instances/<instance_name>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "id": "0",
  "status": "started",
  "links": [
    {
      "rel": "self",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB/instances/0",
      "type": "application/json"
    },
    {
      "rel": "node",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB",
      "type": "application/json"
    },
    {
      "rel": "attribute",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB/instances/0/attributes/ip_address",
      "type": "application/json"
    },
    {
      "rel": "attribute",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB/instances/0/attributes/private_address",
      "type": "application/json"
    },
    {
      "rel": "attribute",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB/instances/0/attributes/public_address",
      "type": "application/json"
    }
  ]
}
```

### Get the attributes list of a given node instance <a name="instance-attributes"></a>

Retrieve the list (as Atom links) of the attributes for this instance.

'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/nodes/<node_name>/instances/<instance_name>/attributes`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "attributes": [
    {
      "rel": "attribute",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB/instances/0/attributes/private_address",
      "type": "application/json"
    },
    {
      "rel": "attribute",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB/instances/0/attributes/public_address",
      "type": "application/json"
    },
    {
      "rel": "attribute",
      "href": "/deployments/6f22a3ef-3ae3-4958-923e-621ab1541677/nodes/ComputeB/instances/0/attributes/ip_address",
      "type": "application/json"
    }
  ]
}

```

### Get the value of an attribute for a given node instance <a name="attribute-value"></a>

Retrieve the value of an attribute for this instance.

'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/nodes/<node_name>/instances/<instance_name>/attributes/<attribute_name>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "name": "ip_address",
  "value": "10.0.0.142"
}
```

### List deployment events <a name="list-events"></a>

Retrieve a list of events. 'Accept' header should be set to 'application/json'.

There are two available endpoints, one allowing to retrieve the events for a given deployment, the other allowing to retrieve the events for all the known deployments.

These endpoints support long polling requests. Long polling is controlled by the `index` and `wait` query parameters.
`wait` allows to specify a polling maximum duration, this is limited to 10 minutes. If not set, the wait time defaults to 5 minutes.
This value can be specified in the form of "10s" or "5m" (i.e., 10 seconds or 5 minutes, respectively). `index` indicates that we are
polling for events newer that this index. A _0_ value will always returns with all currently known event (possibly none if none were
already published), a _1_ value will wait for at least one event.

#### List deployment events concerning a given deployment

`GET    /deployments/<deployment_id>/events?index=1&wait=5m`

#### List all the deployment events

`GET    /events?index=1&wait=5m`

#### Response

A critical note is that the return of these endpoints has no guarantee of new events. It is possible that the timeout was reached before
a new event was published.

Note that the latest index is returned in the JSON structure and as an HTTP Header called `X-yorc-Index`.

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
X-yorc-Index: 1812
```

```json
{
  "events": [
    {"timestamp":"2016-08-16T14:49:25.90310537+02:00","node":"Network","instance":"0","status":"started"},
    {"timestamp":"2016-08-16T14:50:20.712776954+02:00","node":"Compute","instance":"0","status":"started"},
    {"timestamp":"2016-08-16T14:50:20.713890682+02:00","node":"Welcome","instance":"0","status":"initial"},
    {"timestamp":"2016-08-16T14:50:20.7149454+02:00","node":"Welcome","instance":"0","status":"creating"},
    {"timestamp":"2016-08-16T14:50:20.715875775+02:00","node":"Welcome","instance":"0","status":"created"},
    {"timestamp":"2016-08-16T14:50:20.716840754+02:00","node":"Welcome","instance":"0","status":"configuring"},
    {"timestamp":"2016-08-16T14:50:33.355114629+02:00","node":"Welcome","instance":"0","status":"configured"},
    {"timestamp":"2016-08-16T14:50:33.3562717+02:00","node":"Welcome","instance":"0","status":"starting"},
    {"timestamp":"2016-08-16T14:50:54.550463885+02:00","node":"Welcome","instance":"0","status":"started"}
  ],
  "last_index":1812
}
```

### Get latest events index <a name="last-event-idx"></a>

You can retrieve the latest events `index` by using an HTTP `HEAD` request.

`HEAD    /deployments/<deployment_id>/events`

or

`HEAD    /events`

The latest index is returned as an HTTP Header called `X-yorc-Index`.

**Response**:

As per an HTTP `HEAD` request the response as no body.

```HTTP
HTTP/1.1 200 OK
X-yorc-Index: 1812
```

### Get deployment logs <a name="list-logs"></a>

Retrieve a list of logs concerning deployments. 'Accept' header should be set to 'application/json'.

There are two available endpoints, one allowing to retrieve logs for a given deployment, the other allowing to retrieve the logs for all the known deployments.

These endpoints supports long polling requests. Long polling is controlled by the `index` and `wait` query parameters.
`wait` allows to specify a polling maximum duration, this is limited to 10 minutes. If not set, the wait time defaults to 5 minutes.
This value can be specified in the form of "10s" or "5m" (i.e., 10 seconds or 5 minutes, respectively). `index` indicates that we are
polling for events newer that this index. A _0_ value will always returns with all currently known logs (possibly none if none were
already published), a _1_ value will wait for at least one log.

On optional `filter` parameter allows to filters logs by type. Currently available filters are `engine` for yorc deployment logs,
`infrastructure`  for infrastructure provisioning logs and `software` for software provisioning logs. This parameter accepts a coma
separated list of values.

#### Get logs concerning a given deployment

`GET    /deployments/<deployment_id>/logs?index=1&wait=5m&filter=[software, engine, infrastructure]`

#### Get all the logs

`GET    /logs?index=1&wait=5m&filter=[software, engine, infrastructure]`

Note that the latest index is returned in the JSON structure and as an HTTP Header called `X-yorc-Index`.

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
X-yorc-Index: 1781
```

```json
{
    "logs":[
      {"timestamp":"2016-09-05T07:46:09.91123229-04:00","logs":"Applying the infrastructure"},
      {"timestamp":"2016-09-05T07:46:11.663880572-04:00","logs":"Applying the infrastructure"}
     ],
     "last_index":1781
}
```

### Get latest logs index <a name="last-log-idx"></a>

You can retrieve the latest logs `index` by using an HTTP `HEAD` request.

`HEAD    /deployments/<deployment_id>/logs`

or

`HEAD    /logs`

The latest index is returned as an HTTP Header called `X-yorc-Index`.

**Response**:

As per an HTTP `HEAD` request the response as no body.

```HTTP
HTTP/1.1 200 OK
X-yorc-Index: 1812
```

### Get an output <a name="output-value"></a>

Retrieve a specific output. While the deployment status is DEPLOYMENT_IN_PROGRESS an output may be unresolvable in this case an empty string
is returned. With other deployment statuses an unresolvable output leads to an Internal Server Error.

'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/outputs/output_name>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "name":"compute_url",
  "value":"10.197.129.73"
}
```

### List outputs <a name="list-outputs"></a>

Retrieve a list of outputs. 'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/outputs`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "outputs":[
    {"rel":"output","href":"/deployments/5a60975f-e219-4461-b856-8626e6f22d2b/outputs/compute_private_ip","type":"application/json"},
    {"rel":"output","href":"/deployments/5a60975f-e219-4461-b856-8626e6f22d2b/outputs/compute_url","type":"application/json"},
    {"rel":"output","href":"/deployments/5a60975f-e219-4461-b856-8626e6f22d2b/outputs/port_value","type":"application/json"}]
}
```

### Get task information <a name="task-info"></a>

Retrieve information about a task for a given deployment.
'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/tasks/<taskId>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "id": "b4144668-5ec8-41c0-8215-842661520147",
  "target_id": "62d7f67a-d1fd-4b41-8392-ce2377d7a1bb",
  "type": "DEPLOY",
  "status": "DONE"
}
```

### Get task steps information <a name="task-steps-info"></a>

Retrieve information about steps related to a task for a given deployment.
'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/tasks/<taskId>/steps`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
[
    {
        "name": "step1",
        "status": "done"
    },
    {
        "name": "step2",
        "status": "done"
    },
    {
        "name": "step3",
        "status": "error"
    }
]
```

### Update a task step status <a name="task-step-update"></a>

Update a task step status for given deployment and task. For the moment, only step status change from "ERROR" to "DONE" is allowed otherwise an HTTP 401
(Forbidden) error is returned.

`PUT    /deployments/<deployment_id>/tasks/<taskId>/steps/<stepId>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Length: 0
```

### Cancel a task <a name="task-cancel"></a>

Cancel a task for a given deployment. The task should be in status "INITIAL" or "RUNNING" to be canceled otherwise an HTTP 400
(Bad request) error is returned.

`DELETE    /deployments/<deployment_id>/tasks/<taskId>`

**Response**:

```HTTP
HTTP/1.1 202 OK
Content-Length: 0
```

### Resume a task <a name="task-resume"></a>

Resume a task for a given deployment.
The task should be in status "FAILED" to be resumed.
Otherwise an HTTP 400 (Bad request) error is returned.

`PUT    /deployments/<deployment_id>/tasks/<taskId>`

**Response**:

```HTTP
HTTP/1.1 202 OK
Content-Length: 0
```

### Execute a custom command <a name="custom-cmd-exec"></a>

Submit a custom command for a given deployment.

The command corresponds to an operation defined in a node type interface.
The operation is executed by default on all the node type instances.
It can also be applied to a selected subset of instances.
For that, the list of instance identifiers must provided in the request body.

'Content-Type' header should be set to 'application/json'.

`POST    /deployments/<deployment_id>/custom`

Request body allowing to execute command on all the node instances:

```json
{
    "node": "NodeName",
    "name": "Custom_Command_Name",
    "interface": "fully.qualified.interface.name",
    "inputs": {
      "index":"",
      "nb_replicas":"2"
    }
}
```

Request body allowing to execute command on some selected node instances:

```json
{
    "node": "NodeName",
    "name": "Custom_Command_Name",
    "interface": "fully.qualified.interface.name",
    "instances": [ "0", "1" ],
    "inputs": {
      "index":"",
      "nb_replicas":"2"
    }
}
```

**Response**:

```HTTP
HTTP/1.1 202 Accepted
Content-Length: 0
Location: /deployments/08dc9a56-8161-4f54-876e-bb346f1bcc36/tasks/277b47aa-9c8c-4936-837e-39261237cec4
```

### Scale a node <a name="scale-node"></a>

Scales a node on a deployed deployment. A non-zero integer query parameter named `delta` is required and indicates the number of instances to
add or to remove for this scaling operation.

A critical note is that the scaling operation is proceeded asynchronously and a success only guarantees that the scaling operation is successfully
**submitted**.

`POST /deployments/<deployment_id>/scale/<node_name>?delta=<int32>`

A successfully submitted scaling operation will result in an HTTP status code 201 with a 'Location' header relative to the base URI indicating
the URI of the task handling this operation.

```HTTP
HTTP/1.1 201 Created
Location: /deployments/b5aed048-c6d5-4a41-b7ff-1dbdc62c03b0/tasks/012906dc-7916-4529-89b8-fdf628838fe5
Content-Length: 0
```

This endpoint produces no content except in case of error.

This endpoint will failed with an error "400 Bad Request" if:

* another task is already running for this deployment
* the delta query parameter is missing
* the delta query parameter is not an integer or if it is equal to 0

### Execute a workflow <a name="workflow-exec"></a>

Submit a custom workflow for a given deployment.
By adding the optional 'continueOnError' url parameter to your request,
workflow will not stop at the first encountered error and will run to its end.

By default the execution of the workflow's steps take place on all the instances of the workflow's nodes.
It is possible to select instances for the workflow's nodes by adding selection data in the request body.
For nodes that have no selected instances specified, the execution steps take place on all instances.

The request body can also contain input values assignments, that will be provided
to the task handling this workflow execution.

'Content-Type' header should be set to 'application/json'.

`POST /deployments/<deployment_id>/workflows/<workflow_name>[?continueOnError]`

Request body allowing to execute a workflow's steps on selected node instances :

```json
{
    "nodeinstances": [
       {
         "node": "Node_Name",
         "instances": [ "0", "2" ]
       }
    ],
    "inputs": {
      "param1":"",
      "param2":"2"
    }
}
```

A successfully submitted workflow result in an HTTP status code 201 with a 'Location' header relative to the base URI indicating
the URI of the task handling this workflow execution.

**Response**:

```HTTP
HTTP/1.1 201 Created
Content-Length: 0
Location: /deployments/08dc9a56-8161-4f54-876e-bb346f1bcc36/tasks/277b47aa-9c8c-4936-837e-39261237cec4
```

This endpoint will fail with an error "400 Bad Request" if:

* a node specified in request body does not exist
* an instance specified in request body does not exist
* no value is provided in request body for a required workflow input parameter.

### List workflows <a name="list-workflows></a>

Retrieves the list of workflows for a given deployment. 'Accept' header should be set to 'application/json'.

`GET /deployments/<deployment_id>/workflows`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "workflows": [
    {"rel":"workflow","href":"/deployments/55d54226-5ce5-4278-96e4-97dd4cbb4e62/workflows/install","type":"application/json"},
    {"rel":"workflow","href":"/deployments/55d54226-5ce5-4278-96e4-97dd4cbb4e62/workflows/run_job","type":"application/json"},
    {"rel":"workflow","href":"/deployments/55d54226-5ce5-4278-96e4-97dd4cbb4e62/workflows/uninstall","type":"application/json"}
  ]
}
```

### Get workflow description <a name="workflow-info"></a>

Retrieves a JSON representation of a given workflow. 'Accept' header should be set to 'application/json'.

`GET /deployments/<deployment_id>/workflows/<workflow_name>`

**Response**:

```HTTP
HTTP/1.1 200 Created
Content-Type: application/json
```

```json
{
  "Name": "agentsInMaintenance",
  "steps": {
    "ConsulAgent_Maintenance": {
      "node": "ConsulAgent",
      "activity": {
        "call_operation": "custom.maintenance_on"
      }
    }
  }
}
```

## Server related endpoints

These endpoints are related to the queried Yorc server instance.

### Get the Yorc server info

This request return static information about the server. For now it only contains versions information.
`yorc_version` field contains a [semantic versioning](https://semver.org/spec/v2.0.0.html) version number,
while `git_commit` contains the git `sha1` of the server version.
`yorc_version` may contain an option `build identifier` for specific builds like Yorc premium (see below for a sample).

'Accept' header should be set to 'application/json'.

`GET /server/info`

**Response**:

```HTTP
HTTP/1.1 200 Created
Content-Type: application/json
```

```json
{
  "yorc_version": "4.0.0-M10+premium",
  "git_commit": "4572679f61f102088a8fe431fa88fd7f75dd9c1a"
}
```

### Get the Yorc server health

This endpoint is typically used by Consul to check the Yorc service is alive.

'Accept' header should be set to 'application/json'.

`GET /server/health`

**Response**:

```HTTP
HTTP/1.1 200 Created
Content-Type: application/json
```

```json
{
  "value": "passing"
}
```

## Registry

### Get TOSCA Definitions <a name="registry-definitions"></a>

Retrieves the list of the embedded TOSCA definitions and their origins. The origin parameter cloud be `builtin` for yorc builtin definitions or for definitions coming from a plugin it is the name of the plugin binary.

'Accept' header should be set to 'application/json'.

`GET /registry/definitions`

**Response**:

```HTTP
HTTP/1.1 200 Created
Content-Type: application/json
```

```json
{
  "definitions": [
    {"name": "my-custom-types.yml", "origin": "my-custom-plugin"},
    {"name": "normative-types.yml", "origin": "builtin"},
    {"name": "yorc-types.yml", "origin": "builtin"},
    {"name": "yorc-openstack-types.yml", "origin": "builtin"}
  ]
}
```

### Get Delegates Executors <a name="registry-delegates"></a>

Retrieves the list of delegates executors and their origins. The origin parameter cloud be `builtin` for yorc builtin delegates or for delegates coming from a plugin it is the name of the plugin binary.

'Accept' header should be set to 'application/json'.

`GET /registry/delegates`

**Response**:

```HTTP
HTTP/1.1 200 Created
Content-Type: application/json
```

```json
{
  "delegates": [
    {"node_type": "yorc\\.nodes\\.myCustomTypes\\..*", "origin": "my-custom-plugin"},
    {"node_type": "yorc\\.nodes\\.openstack\\..*", "origin": "builtin"}
  ]
}
```

### Get Implementations Executors <a name="registry-implementations"></a>

Retrieves the list of implementations executors and their origins. The origin parameter cloud be `builtin` for yorc builtin implementations or for implementations coming from a plugin it is the name of the plugin binary.

'Accept' header should be set to 'application/json'.

`GET /registry/implementations`

**Response**:

```HTTP
HTTP/1.1 200 Created
Content-Type: application/json
```

```json
{
  "implementations": [
    {"implementation_artifact": "tosca.artifacts.Implementation.MyImplementation", "origin": "my-custom-plugin"},
    {"implementation_artifact": "tosca.artifacts.Implementation.Bash", "origin": "builtin"},
    {"implementation_artifact": "tosca.artifacts.Implementation.Python", "origin": "builtin"},
    {"implementation_artifact": "tosca.artifacts.Implementation.Ansible", "origin": "builtin"}
  ]
}
```

### Get infrastructure usage collectors <a name="registry-infra"></a>

Retrieves the list of infrastructure usage collectors and their origins. The origin parameter cloud be `builtin` for yorc builtin implementations or for implementations coming from a plugin it is the name of the plugin binary.

'Accept' header should be set to 'application/json'.

`GET /registry/infra_usage_collectors`

**Response**:

```HTTP
HTTP/1.1 200 Created
Content-Type: application/json
```

```json
{
    "infrastructures": [
        {
            "id": "slurm",
            "origin": "builtin"
        }
    ]
}
```

## Hosts Pool

### Add a Host to a hosts pool location <a name="hostspool-add"></a>

Adds a host to a hosts pool location managed by this yorc cluster.
The connection object of the JSON request is mandatory while the labels list is optional.
This labels list should be composed with elements with the "op" parameter set to "add" but it could be omitted.

'Content-Type' header should be set to 'application/json'.

`PUT /hosts_pool/<location>/<hostname>`

**Request body**:

```json
{
    "connection": {
        "host": "defaults_to_<hostname>",
        "user": "defaults_to_root",
        "port": "defaults_to_22",
        "private_key": "one_of_password_or_private_key_required",
        "password": "one_of_password_or_private_key_required"
    },
    "labels": [
        {"name": "os", "value": "linux"},
        {"op": "add", "name": "memory", "value": "4G"}
    ]
}
```

**Response**:

```HTTP
HTTP/1.1 201 Created
```

Other possible response response codes are `400` if a host with the same `<hostname>` already exists or if  required parameters are missing.

### Update a Host of the pool <a name="hostspool-update"></a>

Updates labels list or connection of a host of a hosts pool location managed by this yorc cluster.

Both connection and labels list object of the JSON request are optional.
This labels list should be composed with elements with the "op" parameter set to "add" or "remove" but defaults to "add" if omitted. *Adding* a tag that already exists replace its value.

'Content-Type' header should be set to 'application/json'.

`PATCH /hosts_pool/<location>/<hostname>`

**Request body**:

```json
{
    "connection": {
        "password": "new_pass"
    },
    "labels": [
        {"op": "remove", "name": "os", "value": "linux"},
        {"op": "add", "name": "memory", "value": "4G"}
    ]
}
```

**Response**:

```HTTP
HTTP/1.1 200 OK
```

Other possible response response codes are `404` if the host doesn't exist in the pool or `400` if required parameters are missing.

### Delete a Host from the pool <a name="hostspool-delete"></a>

Deletes a host from the hosts pool managed by this yorc cluster.

`DELETE /hosts_pool/<location>/<hostname>`

**Response**:

```HTTP
HTTP/1.1 200 OK
```

Other possible response response codes are `404` if the host doesn't exist in the pool.

### List Hosts in the pool <a name="hostspool-list"></a>

Lists hosts of an hosts pool location managed by this yorc cluster.

'Accept' header should be set to 'application/json'.

`GET /hosts_pool/<location>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "checkpoint": 123,
  "hosts": [
    {"rel":"host","href":"/hosts_pool/location1/host1","type":"application/json"},
    {"rel":"host","href":"/hosts_pool/location1/host2","type":"application/json"}
  ],
  "warnings": ["filter error for host3", "filter error for host4"]
}
```

### Get Host in a hosts pool location <a name="hostspool-get"></a>

Gets the description of a host of an hosts pool location managed by this yorc cluster.

'Accept' header should be set to 'application/json'.

`GET /hosts_pool/<location>/<hostname>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "name": "host1",
  "connection": {
    "user": "ubuntu",
    "password": "newPass",
    "host": "host1",
    "port": 22
  },
  "status": "allocated",
  "message": "allocated for node instance \"Compute-0\" in deployment \"myDeployment\"",
  "labels": {
    "memory": "4G",
    "os": "linux"
  },
  "links": [
    {
      "rel": "self",
      "href": "/hosts_pool/host1",
      "type": "application/json"
    }
  ]
}
```
### Apply Hosts Pool configuration <a name="hostspool-apply"></a>

Applies a Hosts Pool configuration on a specified location. The checkpoint query parameter value is provided in the result of a previous call to the [Hosts Pool List API](#hostspool-list).

'Content-Type' header should be set to 'application/json'.

`POST /hosts_pool/<location>?checkpoint=<uint64>`

**Request body**:

```json
{
    "hosts": [
        {
            "Name": "host1",
            "connection": {
                "user": "test",
                "private_key": "/path/to/secrets/id_rsa",
                "host": "host1.example.com",
                "port": 22
            },
            "labels": {
                "environment": "dev",
                "host.cpu_frequency": "3 GHz",
                "host.disk_size": "150 GB",
                "host.mem_size": "4GB",
                "host.num_cpus": "8",
                "os.architecture": "x86_64",
                "os.distribution": "ubuntu",
                "os.type": "linux",
                "os.version": "17.1"
            }
        },
        {
            "Name": "host2",
            "connection": {
                "user": "test",
                "private_key": "/path/to/secrets/id_rsa",
                "host": "host2.example.com"
            }
        }
    ]
}
```

**Response**:

```HTTP
HTTP/1.1 201 Created
Content-Length: 0

```

To bypass checkpoint verification, the following request can be executed:

'Content-Type' header should be set to 'application/json'.

`PUT /hosts_pool/<location>`

**Request body**:

```json
{
    "hosts": [
        {
            "Name": "host1",
            "connection": {
                "user": "test",
                "private_key": "/path/to/secrets/id_rsa",
                "host": "host1.example.com",
                "port": 22
            },
            "labels": {
                "environment": "dev",
                "host.cpu_frequency": "3 GHz",
                "host.disk_size": "150 GB",
                "host.mem_size": "4GB",
                "host.num_cpus": "8",
                "os.architecture": "x86_64",
                "os.distribution": "ubuntu",
                "os.type": "linux",
                "os.version": "17.1"
            }
        },
        {
            "Name": "host2",
            "connection": {
                "user": "test",
                "private_key": "/path/to/secrets/id_rsa",
                "host": "host2.example.com"
            }
        }
    ]
}
```

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Length: 0
```

Another possible response response code is `400` if the requets body is not correct.

### List Hosts pool locations managed by yorc <a name="hostspool-location"></a>

Lists hosts pool locations managed by this yorc cluster.

'Accept' header should be set to 'application/json'.

`GET /hosts_pool`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "locations": [
    "locationOne",
    "locationTwo",
    "locationThree"
  ]
}
```

## Infrastructure Usage

### Execute a query to retrieve infrastructure usage for a defined infrastructure usage collector <a name="infra-usage-query-exec"></a>

Submit a query for a given infrastructure and a given location to retrieve usage information.
Query parameters can be specified, depending on the infrastructure usage collector implementation.

'Content-Type' header should be set to 'application/json'.

`POST    /infra_usage/<infra_name>/<location_name>[?param1=value1&param2=value2...]`

**Response**:

```HTTP
HTTP/1.1 202 Accepted
Content-Length: 0
Location: /infra_usage/<infra_name>/<location_name>/tasks/<task_id>
```

### Get query information <a name="task-info"></a>

Retrieve information about a task for a given infrastructure usage collector on a given location.
'Accept' header should be set to 'application/json'.

`GET    /infra_usage/<infra_name>/<location_name>/tasks/<taskId>`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
    "id": "9eb9dd64-c08b-45b2-baae-8c657ce33403",
    "target_id": "infra_usage:slurm",
    "type": "Query",
    "status": "DONE",
    "result_set": {
        "cluster": {
            "cpus": {
            "allocated": "90",
            "cpu_load": "0.01-0.03",
            "idle": "78",
            "other": "8",
            "total": "176"
        },
        "memory": {
            "allocated": "242,464 MB",
            "total": "360,448 MB"
            },
        "nodes": "22"
        },
        "partitions": [
        {
            "cpus": {
                "allocated": "90",
                "cpu_load": "0.01-0.03",
                "idle": "38",
                "other": "8",
                "total": "136"
            },
            "jobs": {
                "pending": "0",
                "running": "48"
            },
            "memory": {
                "allocated": "242,464 MB",
                "total": "278,528 MB"
            },
            "name": "debug",
            "nodes": "17",
            "nodes_list": "hpda[1-2,5-17,19-20,23-25]",
            "state": "up"
        }]
    }
}
```

### Delete a query <a name="query-delete"></a>

Delete an existing query. The task should be in status "DONE" or "FAILED" to be deleted otherwise an HTTP 400
(Bad request) error is returned.

`DELETE    /infra_usage/<infra_name>/<location_name>/tasks/<taskId>`

**Response**:

```HTTP
HTTP/1.1 202 OK
Content-Length: 0
```
### List all queries about an infrastructure usage <a name="list-query"></a>

Retrieve all queries run about infrastructure usage.
'Accept' header should be set to 'application/json'.

`GET    /infra_usage?target=slurm`

**Response**
```
HTTP/1.1 200 OK
Content-Type: application/json
```
```json
{
"tasks": [
  {
    "rel": "task",
    "href": "/infra_usage/slurm/myLocationOne/tasks/b0d91e99-1970-4fc0-8cc2-2cb4f5007e27",
    "type": "application/json"
  },
  {
    "rel": "task",
    "href": "/infra_usage/slurm/myLocationOne/tasks/b4f9682d-7020-45dc-b2db-d0c84b8c30e2",
    "type": "application/json"
  }
 ]
}
```
## Locations ##

### List locations

List all the existent location definitions.

'Content-Type' header should be set to 'application/json'.

`GET    /locations`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "locations":[
    {
      "rel": "location",
      "href": "/locations/location1",
      "type": "application/json"
    }
  ]
}
```

### Get location

Get an existent location's definitions.

'Content-Type' header should be set to 'application/json'.

`GET    /locations/location1`

**Response**:

```HTTP
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "name": "location1",
  "type": "openstack",
  "properties": {
    "auth_url":"http://openstack:5000/v2.0",
    "os_default_security_groups":["default","lax"],
    "password":"StPass","private_network_name":"private-test",
    "public_network_name":"not_supported","region":"RegionOne",
    "tenant_id":"use_tid_or_tname",
    "tenant_name":"Tname",
    "user_name":"Starlings"
  }
}
```

### Create location <a name="location-create"></a>

Create a new location.

'Content-Type' header should be set to 'application/json'.

`PUT /locations/<location_name>`

**Request body**:

```json
{
  "type": "slurm",
  "properties": {
		"user_name":   "slurmuser1",
		"private_key": "/path/to/key",
		"url":         "10.1.2.3",
		"port":        22
  }
}
```

**Response**:

```HTTP
HTTP/1.1 201 Created
```

Other possible response response code is `400` if a location with the same `<location_name>` already exists.

### Update location

Update an existent location.

'Content-Type' header should be set to 'application/json'.

`PATCH /locations/<location_name>`

**Request body**:

```json
{
  "type": "slurm",
  "properties": {
		"user_name":   "slurmuser2",
		"private_key": "/path/to/key",
		"url":         "10.1.2.3",
		"port":        22
  }
}
```

**Response**:

```HTTP
HTTP/1.1 200 OK
```

Other possible response response code is `400` if a location with the `<location_name>` does not exist.

### Remove location

Remove a location with a given name.

'Content-Type' header should be set to 'application/json'.

`DELETE /locations/<location_name>`

**Response**:

```HTTP
HTTP/1.1 200 OK
```

Other possible response response code is `400` if a location with the name `<location_name>` does not exist.

