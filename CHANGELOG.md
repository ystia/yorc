# Yorc Changelog

## 3.1.1 (February 06, 2019)

### BUG FIXES

* Purging n deployment in parallel, one can fail on error: Missing targetId for task with id ([GH-293](https://github.com/ystia/yorc/issues/293))
* Can't deploy applications using a secured yorc/consul ([GH-274](https://github.com/ystia/yorc/issues/274))
* Unable to purge an application that appears in the list ([GH-238](https://github.com/ystia/yorc/issues/238))
* K8S jobs namespace should not be removed if its provided ([GH-245](https://github.com/ystia/yorc/issues/245))
* Deployment with a topology parsing error remains in initial status ([GH-283](https://github.com/ystia/yorc/issues/283)
* Interface name is not retrieved from custom command Rest request ([GH-287](https://github.com/ystia/yorc/issues/287)
* Instances are adding into topology before creating task ([GH-289](https://github.com/ystia/yorc/issues/289)
* All ssh connections to Slurm are killed if ssh server has reached the max number of allowed sessions ([GH-291](https://github.com/ystia/yorc/issues/291)
* Missing events for uninstall workflow in purge task ([GH-302](https://github.com/ystia/yorc/issues/302)
* It can take a considerable delay for a deployment to change status to UNDEPLOYMENT_IN_PROGRESS ([GH-306](https://github.com/ystia/yorc/issues/306)
* Slurm job monitoring is not designed for concurrency ([GH-308](https://github.com/ystia/yorc/issues/308)

## 3.1.0 (December 20, 2018)

### BUG FIXES

* When scaling down instances are not cleaned from consul ([GH-257](https://github.com/ystia/yorc/issues/257))
* Yorc bootstrap fails if downloadable URLs are too long ([GH-247](https://github.com/ystia/yorc/issues/247))

### ENHANCEMENTS

* Increase default workers number per Yorc server from `3` to `30` ([GH-244](https://github.com/ystia/yorc/issues/244))

### BUG FIXES

* Bootstrap fails on Red Hat Enterprise Linux 7.5 ([GH-252](https://github.com/ystia/yorc/issues/252))

## 3.1.0-RC2 (December 18, 2018)

### DEPENDENCIES

* Technical update to use Alien4Cloud 2.1.0 final version (Used in bootstrap)

## 3.1.0-RC1 (December 17, 2018)

### ENHANCEMENTS

* Support Jobs lifecycle enhancements (new operations `submit`, `run`, `cancel`) ([GH-196](https://github.com/ystia/yorc/issues/196))
* Forbid the parallel execution of several scheduled actions. This is for instance used for the asynchronous run operation of Jobs. This will prevent a same action to be scheduled in parallel (for jobs it will prevent checking and doing same actions several times) ([GH-230](https://github.com/ystia/yorc/issues/230))
* Generate Alien 2.1-compatible events ([GH-148](https://github.com/ystia/yorc/issues/148))

### BUG FIXES

* No output properties for services on GKE ([GH-214](https://github.com/ystia/yorc/issues/214))
* K8S service IP missing in runtime view when deploying on GKE ([GH-215](https://github.com/ystia/yorc/issues/215))
* Bootstrap of HA setup fails on GCP, at step configuring the NFS Client component ([GH-218](https://github.com/ystia/yorc/issues/218))
* Fix issue when default yorc.pem is used by Ansible with ssh-agent ([GH-233](https://github.com/ystia/yorc/issues/233))
* Publish workflow events when custom workflow is finished ([GH-234](https://github.com/ystia/yorc/issues/234))
* Bootstrap without internet access fails to get terraform plugins for local yorc ([GH-239](https://github.com/ystia/yorc/issues/239))
* CUDA_VISIBLE_DEVICES contains some unwanted unprintable characters [GH-210](https://github.com/ystia/yorc/issues/210))

## 3.1.0-M7 (December 07, 2018)

### DEPENDENCIES

* The orchestrator requires now at least Ansible 2.7.2 (upgrade from 2.6.3 introduced in [GH-194](https://github.com/ystia/yorc/issues/194))

### FEATURES

* Allow to bootstrap a full stack Alien4Cloud/Yorc setup using yorc CLI ([GH-131](https://github.com/ystia/yorc/issues/131))

### ENHANCEMENTS

* Use ssh-agent to not write ssh private keys on disk ([GH-201](https://github.com/ystia/yorc/issues/201))

### BUG FIXES

* ConnectTo relationship not working for kubernetes topologies ([GH-212](https://github.com/ystia/yorc/issues/212))

## 3.1.0-M6 (November 16, 2018)

### FEATURES

* Support GCE virtual private networks (VPC) ([GH-80](https://github.com/ystia/yorc/issues/80))
* Support Kubernetes Jobs. ([GH-86](https://github.com/ystia/yorc/issues/86))

### ENHANCEMENTS

* Allow user to provide an already existing namespace to use when creating Kubernetes resources ([GH-76](https://github.com/ystia/yorc/issues/76))

### BUG FIXES

* Generate unique names for GCP resources ([GH-177](https://github.com/ystia/yorc/issues/177))
* Need a HOST public_ip_address attribute on Hosts Pool compute nodes ([GH-199](https://github.com/ystia/yorc/issues/199))

## 3.1.0-M5 (October 26, 2018)

### FEATURES

* Support GCE Block storages. ([GH-82](https://github.com/ystia/yorc/issues/81))

### ENHANCEMENTS

* Concurrent workflows and custom commands executions are now allowed except when a deployment/undeployment/scaling operation is in progress ([GH-182](https://github.com/ystia/yorc/issues/182))
* Enable scaling of Kubernetes deployments ([GH-77](https://github.com/ystia/yorc/issues/77))

## 3.1.0-M4 (October 08, 2018)

### DEPENDENCIES

* The orchestrator requires now at least Terraform 0.11.8 and following Terraform plugins (with corresponding version constraints): `Consul (~> 2.1)`, `AWS (~> 1.36)`, `OpenStack (~> 1.9)`, `Google (~ 1.18)` and `null provider (~ 1.0)`. (Terraform upgrade from 0.9.11 introduced in [GH-82](https://github.com/ystia/yorc/issues/82))
* Consul version updated to 1.2.3

### FEATURES

* Support GCE Public IPs. ([GH-82](https://github.com/ystia/yorc/issues/82))

### IMPROVEMENTS

* Split workflow execution unit to step in order to allow a unique workflow to be executed by multiple Yorc instances. ([GH-93](https://github.com/ystia/yorc/issues/93))
* Make the run step of a Job execution asynchronous not to block a worker during the duration of the job. ([GH-85](https://github.com/ystia/yorc/issues/85))
* Added configuration parameters in Kubernetes infrastructure allowing to connect from outside to a cluster created on Google Kubernetes Engine ([GH-162](https://github.com/ystia/yorc/issues/162))
* Allow to upgrade from version 3.0.0 to a newer version without loosing existing data ([GH-130](https://github.com/ystia/yorc/issues/130))

### BUG FIXES

* Inputs are not injected into Slurm (srun) jobs ([GH-161](https://github.com/ystia/yorc/issues/161))
* Yorc consul service registration fails if using TLS ([GH-153](https://github.com/ystia/yorc/issues/153))
* Retrieving operation output when provisioning several instances resolves to the same value for all instances even if they are actually different ([GH-171](https://github.com/ystia/yorc/issues/171))

## 3.1.0-M3 (September 14, 2018)

### IMPROVEMENTS

* Allow to use 'In cluster' authentication when Yorc is deployed on Kubernetes. This allows to use credentials provided by Kubernetes itself. ([GH-156](https://github.com/ystia/yorc/issues/156))

### BUG FIXES

* REQ_TARGET keyword into TOSCA functions was broken. This was introduced during the upgrade to Alien4Cloud 2.0 that changed how requirements definition on node templates ([GH-159](https://github.com/ystia/yorc/issues/159))
* Parse Alien specific way of defining properties on relationships ([GH-155](https://github.com/ystia/yorc/issues/155))

## 3.1.0-M2 (August 24, 2018)

### DEPENDENCIES

* The orchestrator requires now at least Ansible 2.6.3 (upgrade from 2.4.1 introduced in [GH-146](https://github.com/ystia/yorc/issues/146))

### IMPROVEMENTS

* Providing Ansible task output in Yorc logs as soon as the task has finished ([GH-146](https://github.com/ystia/yorc/issues/146))

### BUG FIXES

* Parse of TOSCA value assignment literals as string. This prevents issues on strings being interpreted as float and rounded when converted back into strings (GH-137)
* Install missing dependency `jmespath` required by the `json_query` filter of Ansible (GH-139)
* Capabilities context props & attribute are not injected anymore for Ansible recipes implementation (GH-141)

## 3.1.0-M1 (August 6, 2018)

### FEATURES

* Manage applications secrets (GH-134)

### IMPROVEMENTS

* Relax errors on TOSCA get_attributes function resolution that may produce empty results instead of errors (GH-75)

### BUG FIXES

* Fix build issues on go1.11 (GH-72)

## 3.0.0 (July 11, 2018)

### Naming & Open Source community

Yorc 3.0.0 is the first major version since we open-sourced the formerly known Janus project. Previous versions have been made available on GitHub.

We are still shifting some of our tooling like road maps and backlogs publicly available tools. The idea is to make project management clear and to open Yorc to external contributions.

### Shifting to Alien4Cloud 2.0

Alien4Cloud released recently a fantastic major release with new features leveraged by Yorc to deliver a great orchestration solution.

Among many features, the ones we will focus on below are:

* UI redesign: Alien4Cloud 2.0.0 includes various changes in UI in order to make it more consistent and easier to use.
* Topology modifiers: Alien4Cloud 2.0.0 allows to define modifiers that could be executed in various phases prior to the deployment. Those modifiers allow to transform a given TOSCA topology.

### New GCP infrastructure

We are really excited to announce our first support of [Google Cloud Platform](https://cloud.google.com/).

Yorc now natively supports [Google Compute Engine](https://cloud.google.com/compute/) to create compute on demand on GCE.

### New Hosts Pool infrastructure

Yorc 3.0.0 supports a new infrastructure that we called "Hosts Pool". It allows to register generic hosts into Yorc and let Yorc allocate them for deployments. These hosts can be anything, VMs, physical machines, containers, ... whatever as long as we can ssh into them for provisioning. Yorc exposes a REST API and a CLI that allow to manage the hosts pool, making it easy to integrate it with other tools.

For more informations about the Hosts Pool infrastructure, check out [our dedicated documentation](http://yorc.readthedocs.io/en/latest/infrastructures.html#hosts-pool).

### Slurm infrastructure

We made some improvements with our Slurm integration:

* We now support Slurm "features" (which are basically tags on nodes) and "constraints" syntax to allocate nodes. [Examples here](https://wiki.rc.usf.edu/index.php/SLURM_Using_Features_and_Constraints).
* Support of srun and sbatch commands (see Jobs scheduling below)

### Refactoring Kubernetes infrastructure support

In Yorc 2 we made a first experimental integration with Kubernetes. This support and associated TOSCA types are deprecated in Yorc 3.0. Instead we switched to [new TOSCA types defined collectively with Alien4Cloud](http://alien4cloud.github.io/#/documentation/2.0.0/orchestrators/kubernetes/kubernetes_walkthrough.html).

This new integration will allow to build complex Kubernetes topologies.

### Support of Alien4Cloud Services

[Alien4Cloud has a great feature called "Services"](http://alien4cloud.github.io/common/features.html#/documentation/2.0.0/concepts/services.html). It allows both to define part of an application to be exposed as a service so that it can be consumed by other applications, or to register an external service in Alien4Cloud to be exposed and consumed by applications.

This feature allows to build new use cases like cross-infrastructure deployments or shared services among many others.

We are very excited to support it!

### Operations on orchestrator's host

Yet another super interesting feature! Until now TOSCA components handled by Yorc were designed to be hosted on a compute (whatever it was) that means that component's life-cycle scripts were executed on the provisioned compute. This feature allows to design components that will not necessary be hosted on a compute, and if not, life-cycle scripts are executed on the Yorc's host.

This opens a wide range of new use cases. You can for instance implement new computes implementations in pure TOSCA by calling cloud-providers CLI tools or interact with external services

Icing on the cake, for security reasons those executions are by default sand-boxed into containers to protect the host from mistakes and malicious usages.

### Jobs scheduling

This release brings a tech preview support of jobs scheduling. It allows to design workloads made of Jobs that could interact with each other and with other "standard" TOSCA component within an application. We worked hard together with the Alien4Cloud team to extent TOSCA to support Jobs scheduling.

In this release we mainly focused on the integration with Slurm for supporting this feature (but we are also working on Kubernetes for the next release :smile:). Bellow are new supported TOSCA types and implementations:

* SlurmJobs: will lead to issuing a srun command with a given executable file.  
* SlurmBatch: will lead to issuing a sbatch command with a given batch file and associated executables
* Singularity integration: allows to execute a Singularity container instead of an executable file.

### Mutual SSL auth between Alien & Yorc

Alien4Cloud and Yorc can now mutually authenticate themselves with TLS certificates.

### New Logs formating

We constantly try to improve feedback returned to our users about runtime execution. In this release we are publishing logs with more context on the node/instance/operation/interface to which the log relates to.

### Monitoring

Yorc 3.0 brings foundations on applicative monitoring, it allows to monitor compute liveness at a interval defined by the user. When a compute goes down or up we use or events API to notify the user and Alien4Cloud to monitor an application visually within the runtime view.

Our monitoring implementation was designed to be a fault-tolerant service.
