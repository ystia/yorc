# Janus HTTP (REST) API

Janus runs an HTTP server that exposes an API in a restful manner.
Currently supported urls are:

## Deployments

### Submit a CSAR to deploy
Creates a new deployment by uploading a CSAR. 'Content-Type' header should be set to 'application/zip'.

```POST /deployments```

A successfully submitted deployment will result in an HTTP status code 201 with a 'Location' header relative to the base URI indicating the
deployment URI.

```
HTTP/1.1 201 Created
Location: /deployments/b5aed048-c6d5-4a41-b7ff-1dbdc62c03b0
Content-Length: 0
```

This endpoint produces no content except in case of error.

A critical note is that the deployment is proceeded asynchronously and a success only guarantees that the deployment is successfully
**submitted**.

### List deployments

Retrieves the list of deployments. 'Accept' header should be set to 'application/json'.

```GET /deployments```

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

### Undeploy  an active deployment

Undeploy a deployment

`DELETE /deployments/<deployment_id>`

```
HTTP/1.1 202 Accepted
Content-Length: 0
```

This endpoint produces no content except in case of error.

### Get the deployment status

Retrieve the deployment status. 'Accept' header should be set to 'application/json'.

```GET    /deployments/<deployment_id>```

**Response**

```
HTTP/1.1 200 OK
Content-Type: application/json
```
```json
{"id":"55d54226-5ce5-4278-96e4-97dd4cbb4e62","status":"DEPLOYED"}
```

### Get the deployment status of a node

Retrieve the deployment status. 'Accept' header should be set to 'application/json'.

```GET    /deployments/<deployment_id>/node/<node_name>/events```

**Response**

```
HTTP/1.1 200 OK
Content-Type: application/json
```
```json
{"status":"started"}
```


### List deployment events

Retrieve a list of events. 'Accept' header should be set to 'application/json'.
This endpoint supports long polling requests. Long polling is controlled by the `index` and `wait` query parameters.
`wait` allows to specify a polling maximum duration, this is limited to 10 minutes. If not set, the wait time defaults to 5 minutes.
This value can be specified in the form of "10s" or "5m" (i.e., 10 seconds or 5 minutes, respectively). `index` indicates that we are
polling for events newer that this index. A _0_ value will always returns with all currently known event (possibly none if none were
already published), a _1_ value will wait for at least one event.

```GET    /deployments/<deployment_id>/events?index=1&wait=5m```

A critical note is that the return of this endpoint is no guarantee of new events. It is possible that the timeout was reached before
a new event was published.

**Response**
```
HTTP/1.1 200 OK
Content-Type: application/json
```
```json
{
  "events": [
    {"timestamp":"2016-08-16T14:49:25.90310537+02:00","node":"Network","status":"started"},
    {"timestamp":"2016-08-16T14:50:20.712776954+02:00","node":"Compute","status":"started"},
    {"timestamp":"2016-08-16T14:50:20.713890682+02:00","node":"Welcome","status":"initial"},
    {"timestamp":"2016-08-16T14:50:20.7149454+02:00","node":"Welcome","status":"creating"},
    {"timestamp":"2016-08-16T14:50:20.715875775+02:00","node":"Welcome","status":"created"},
    {"timestamp":"2016-08-16T14:50:20.716840754+02:00","node":"Welcome","status":"configuring"},
    {"timestamp":"2016-08-16T14:50:33.355114629+02:00","node":"Welcome","status":"configured"},
    {"timestamp":"2016-08-16T14:50:33.3562717+02:00","node":"Welcome","status":"starting"},
    {"timestamp":"2016-08-16T14:50:54.550463885+02:00","node":"Welcome","status":"started"}
  ],
  "last_index":1812
}
```

### Get logs of applications

Retrieve the deployment status. 'Accept' header should be set to 'application/json'.

```GET    /deployments/<deployment_id>/logs?index=1&wait=5m&filter=[software, engine, infrastructure]```

**Response**

```
HTTP/1.1 200 OK
Content-Type: application/json
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
