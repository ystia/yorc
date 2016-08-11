# Janus HTTP (REST) API

Janus runs an HTTP server that exposes an API in a restful manner.
Currently supported urls are:

**Deployments**
- `POST   /deployments` : Creates a new deployment by uploading a CSAR. 'Content-Type' header should be set to 'application/zip'.
- `GET    /deployments` : Retrieve the list of deployments. 'Accept' header should be set to 'application/json'.
- `DELETE /deployments/<deployment_id>` : Undeploy a deployment.
- `GET    /deployments/<deployment_id>` : Retrieve the deployment status. 'Accept' header should be set to 'application/json'.
- `GET    /deployments/<deployment_id>/events?index=1&wait=5m` : Retrieve a list of events. 'Accept' header should be set to 'application/json'.
  This endpoint supports long polling requests. Long polling is controlled by the `index` and `wait` query parameters.
  `wait` allows to specify a polling maximum duration, this is limited to 10 minutes. If not set, the wait time defaults to 5 minutes.
  This value can be specified in the form of "10s" or "5m" (i.e., 10 seconds or 5 minutes, respectively). `index` indicates that we are
  polling for events newer that this index. A _0_ value will always returns with all currently known event (possibly none if none were
  already published), a _1_ value will wait for at least one event. A critical note is that the return of this endpoint is no guarantee of
  new events. It is possible that the timeout was reached before a new event was published.

