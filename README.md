# Ystia Orchestrator

<!--
TODO: add badges (travis/gocodereport/...)
-->
[![Build Status](https://travis-ci.org/ystia/yorc.svg?branch=develop)](https://travis-ci.org/ystia/yorc) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)

Yorc is an hybrid cloud/HPC [TOSCA](http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html) orchestrator.

It aims to support the whole application lifecycle, from deployment, scaling, monitoring, self-healing, self-scaling to application upgrade, over hybrid infrastructures (IaaS, HPC schedulers, CaaS).

Yorc is TOSCA native to allow handling complex applications in a standard way. Yorc is also workflow-driven,
this means that it doesn't contain any hard-coded lifecycle logic. This allows to fully customize applications behavior and to execute custom workflows at runtime.

Yorc is designed for large-scale, it is built with a tasks / stateless workers model in mind allowing to scale it
horizontally easily.

Finally, while you can easily interact with Yorc directly thanks to its comprehensive REST API and a modern CLI,
the recommended way to use Yorc is to model your applications in a powerful TOSCA designer called [alien4cloud](https://alien4cloud.github.io) and to use it to deploy and interact with your application at runtime. To do it
we developed an [Alien4Cloud plugin for Yorc](https://github.com/ystia/yorc-a4c-plugin)

## How to download the Ystia Orchestrator

Yorc releases can be downloaded from the [github project releases page](https://github.com/ystia/yorc/releases).

## How to contribute to this project

We warmly welcome any kind of contribution from feedbacks and constructive criticism to code changes.
Please read our [contribution guidelines](CONTRIBUTING.md) for more information.

<!--
TODO: link to readthedoc.org

## Documentation
-->

## Project History

This work was originally developed by _Bull Atos Technologies_ under the project code name _Janus_. The project name changed to __Ystia Orchestrator **(Yorc)**__ during the version 3.0 development cycle.

## Licensing

Yorc is licensed under the [Apache 2.0 License](LICENSE).
