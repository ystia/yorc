# Ystia Orchestrator

[![Download](https://img.shields.io/badge/download-v4.2.0--milestone.1-blue)](https://github.com/ystia/yorc/releases/tag/v4.4.0-milestone.1) [![Build Status](https://github.com/ystia/yorc/actions/workflows/build.yml/badge.svg?branch=develop)](https://github.com/ystia/yorc/actions) [![Documentation Status](https://readthedocs.org/projects/yorc/badge/?version=latest)](http://yorc.readthedocs.io/en/latest/?badge=latest) [![Go Report Card](https://goreportcard.com/badge/github.com/ystia/yorc)](https://goreportcard.com/report/github.com/ystia/yorc) [![license](https://img.shields.io/github/license/ystia/yorc.svg)](https://github.com/ystia/yorc/blob/develop/LICENSE) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com) [![Docker Pulls](https://img.shields.io/docker/pulls/ystia/yorc.svg?style=flat)](https://hub.docker.com/r/ystia/yorc) [![Join the chat at https://gitter.im/ystia/yorc](https://badges.gitter.im/ystia/yorc.svg)](https://gitter.im/ystia/yorc?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

<p align="center">
  <img src="https://raw.githubusercontent.com/ystia/yorc/develop/doc/_static/logo/github-repository-open-graph-template.png" width="50%" height="50%" title="Yorc Logo">
</p>

---

Yorc is an hybrid cloud/HPC [TOSCA](http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html) orchestrator.

It aims to support the whole application lifecycle, from deployment, scaling, monitoring, self-healing, self-scaling to application upgrade, over hybrid infrastructures (IaaS, HPC schedulers, CaaS).

Yorc is TOSCA native to allow handling complex applications in a standard way. Yorc is also workflow-driven,
this means that it doesn't contain any hard-coded lifecycle logic. This allows to fully customize applications behavior and to execute custom workflows at runtime.

Yorc is designed for large-scale, it is built with a tasks / stateless workers model in mind allowing to scale it
horizontally easily.

Finally, while you can easily interact with Yorc directly thanks to its comprehensive REST API and a modern CLI,
the recommended way to use Yorc is to model your applications in a powerful TOSCA designer called [alien4cloud](https://alien4cloud.github.io) and to use it to deploy and interact with your application at runtime.
Yorc is now the official orchestrator for Alien4Cloud and Alien4Cloud distributions comes with a specific plugin for Yorc. Sources of this plugin could be found here <https://github.com/alien4cloud/alien4cloud-yorc-provider>

## How to download the Ystia Orchestrator

Yorc releases can be downloaded from our [GitHub Release](https://github.com/ystia/yorc/releases).

Grab the [latest release here](https://github.com/ystia/yorc/releases/latest).

Docker images could be found on [Docker Hub](https://hub.docker.com/r/ystia/yorc).

## How to contribute to this project

We warmly welcome any kind of contribution from feedbacks and constructive criticism to code changes.
Please read our [contribution guidelines](CONTRIBUTING.md) for more information.

## Documentation

The project documentation is available on [readthedocs](http://yorc.readthedocs.io/en/latest/)

## Project History

This work was originally developed by _Bull Atos Technologies_ under the project code name _Janus_. The project name changed to __Ystia Orchestrator **(Yorc)**__ during the version 3.0 development cycle.

## Licensing

Yorc is licensed under the [Apache 2.0 License](LICENSE).
