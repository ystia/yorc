# Yorc CI Integration Tests Runner

This repository contains elements needed to run Yorc integration tests in a Jenkins CI.

The [`Jenkinsfile`](./Jenkinsfile) describe the CI pipeline.

The [`Dockerfile`](./Dockerfile) allows to build a docker image that contains every things needed to execute our godog tests.
