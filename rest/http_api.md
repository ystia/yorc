# Janus HTTP (REST) API

Janus runs an HTTP server that exposes an API in a restful manner.
Currently supported urls are:

## Deployments

Adding the 'pretty' url parameter to your requests allow to generate an indented json output. 

### Submit a CSAR to deploy <a name="submit-csar"></a>
Creates a new deployment by uploading a CSAR. 'Content-Type' header should be set to 'application/zip'.

There is two ways to submit a new deployment, you can let Janus generate a unique deployment ID or you can specify it

#### Auto-generated deployment ID
In this case you should use a `POST` method.

`POST /deployments`


#### Customized deployment ID
In this case you should use a `PUT` method. There is some constraints on submitting a deployment with a given ID:
  * This ID should respect the following format: `^[-_0-9a-zA-Z]+$` and be less than 36 characters long (otherwise a `400 BadRequest` error is returned)
  * This ID should not already be in used (otherwise a `409 Conflict` error is returned)

`PUT /deployments/<deployment_id>`


#### Result

In both submission ways, a successfully submitted deployment will result in an HTTP status code 201 with a 'Location' header relative to the base URI indicating the task URI handling the deployment process.

```
HTTP/1.1 201 Created
Location: /deployments/b5aed048-c6d5-4a41-b7ff-1dbdc62c03b0/tasks/b4144668-5ec8-41c0-8215-842661520147
Content-Length: 0
```

This endpoint produces no content except in case of error.

A critical note is that the deployment is proceeded asynchronously and a success only guarantees that the deployment is successfully
**submitted**.

### List deployments <a name="list-deps"></a>

Retrieves the list of deployments. 'Accept' header should be set to 'application/json'.

`GET /deployments`

**Response**

```
HTTP/1.1 200 OK
Content-Type: application/json
```
```json
{
  "deployments": [
    {"rel":"deployment","href":"/deployments/55d54226-5ce5-4278-96e4-97dd4cbb4e62","type":"application/json"}
  ]
}
```

### Undeploy  an active deployment <a name="undeploy"></a>

Undeploy a deployment. By adding the optional 'purge' url parameter to your request you will suppress any reference to this deployment from the Janus database at the end of the undeployment. A successful call to this endpoint results in a HTTP status code 202 with a 'Location' header relative to the base URI indicating the task URI handling the undeployment process.

`DELETE /deployments/<deployment_id>[?purge]`

```
HTTP/1.1 202 Accepted
Location: /deployments/b5aed048-c6d5-4a41-b7ff-1dbdc62c03b0/tasks/b4144668-5ec8-41c0-8215-842661520147
Content-Length: 0
```

This endpoint produces no content except in case of error.

A critical note is that the undeployment is proceeded asynchronously and a success only guarantees that the undeployment task is successfully
**submitted**.


### Get the deployment information <a name="dep-info"></a>

Retrieve the deployment status and the list (as Atom links) of the nodes and tasks related the deployment.

'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>`

**Response**

```
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

**Response**

```
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

Retrieve the node instance status and the list (as Atom links) of the attributes for this instance.
 
'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/nodes/<node_name>/instances/<instance_name>`

**Response**

```
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

**Response**

```
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

Retrieve the value an attributes for this instance.
 
'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/nodes/<node_name>/instances/<instance_name>/attributes/<attribute_name>`

**Response**

```
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
This endpoint supports long polling requests. Long polling is controlled by the `index` and `wait` query parameters.
`wait` allows to specify a polling maximum duration, this is limited to 10 minutes. If not set, the wait time defaults to 5 minutes.
This value can be specified in the form of "10s" or "5m" (i.e., 10 seconds or 5 minutes, respectively). `index` indicates that we are
polling for events newer that this index. A _0_ value will always returns with all currently known event (possibly none if none were
already published), a _1_ value will wait for at least one event.

`GET    /deployments/<deployment_id>/events?index=1&wait=5m`

A critical note is that the return of this endpoint is no guarantee of new events. It is possible that the timeout was reached before
a new event was published.

Note that the latest index is returned in the JSON structure and as an HTTP Header called `X-Janus-Index`.

**Response**
```
HTTP/1.1 200 OK
Content-Type: application/json
X-Janus-Index: 1812
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

The latest index is returned as an HTTP Header called `X-Janus-Index`.

**Response**

As per an HTTP `HEAD` request the response as no body.

```
HTTP/1.1 200 OK
X-Janus-Index: 1812
```

### Get logs of a deployment <a name="list-logs"></a>

Retrieve a list of logs. 'Accept' header should be set to 'application/json'.
This endpoint supports long polling requests. Long polling is controlled by the `index` and `wait` query parameters.
`wait` allows to specify a polling maximum duration, this is limited to 10 minutes. If not set, the wait time defaults to 5 minutes.
This value can be specified in the form of "10s" or "5m" (i.e., 10 seconds or 5 minutes, respectively). `index` indicates that we are
polling for events newer that this index. A _0_ value will always returns with all currently known logs (possibly none if none were
already published), a _1_ value will wait for at least one log.


On optional `filter` parameter allows to filters logs by type. Currently available filters are `engine` for Janus deployment logs, 
`infrastructure`  for infrastructure provisioning logs and `software` for software provisioning logs. This parameter accepts a coma 
separated list of values.  

`GET    /deployments/<deployment_id>/logs?index=1&wait=5m&filter=[software, engine, infrastructure]`

Note that the latest index is returned in the JSON structure and as an HTTP Header called `X-Janus-Index`.

**Response**

```
HTTP/1.1 200 OK
Content-Type: application/json
X-Janus-Index: 1781
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

The latest index is returned as an HTTP Header called `X-Janus-Index`.

**Response**

As per an HTTP `HEAD` request the response as no body.

```
HTTP/1.1 200 OK
X-Janus-Index: 1812
```

### Get an output <a name="output-value"></a>

Retrieve a specific output. While the deployment status is DEPLOYMENT_IN_PROGRESS an output may be unresolvable in this case an empty string
is returned. With other deployment statuses an unresolvable output leads to an Internal Server Error. 
 
'Accept' header should be set to 'application/json'.

`GET    /deployments/<deployment_id>/outputs/output_name>`

**Response**
```
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

**Response**
```
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

**Response**
```
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


### Cancel a task <a name="task-cancel"></a>

Cancel a task for a given deployment. The task should be in status "INITIAL" or "RUNNING" to be canceled otherwise an HTTP 400 
(Bad request) error is returned. 

`DELETE    /deployments/<deployment_id>/tasks/<taskId>`

**Response**
```
HTTP/1.1 202 OK
Content-Length: 0
```

### Execute a custom command <a name="custom-cmd-exec"></a>
Submit a custom command for a given deployment.  
'Content-Type' header should be set to 'application/json'.

`POST    /deployments/<deployment_id>/custom`

Request body:
```json
{
    "node": "NodeName",
    "name": "Custom_Command_Name",
    "inputs": {
    	"index":"",
    	"nb_replicas":"2"
    }
}
```


**Response**
```
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

```
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
Submit a custom workflow for a given deployment. By adding the optional 'continueOnError' url parameter to your request workflow will
not stop at the first encountered error and will run to its end. 

`POST /deployments/<deployment_id>/workflows/<workflow_name>[?continueOnError]`

A successfully submitted workflow result in an HTTP status code 201 with a 'Location' header relative to the base URI indicating
the URI of the task handling this workflow execution.


**Response**
```
HTTP/1.1 201 Created
Content-Length: 0
Location: /deployments/08dc9a56-8161-4f54-876e-bb346f1bcc36/tasks/277b47aa-9c8c-4936-837e-39261237cec4
```

### List workflows <a name="list-workflows></a>
Retrieves the list of workflows for a given deployment. 'Accept' header should be set to 'application/json'.

`GET /deployments/<deployment_id>/workflows`

**Response**

```
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

**Response**
```
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
