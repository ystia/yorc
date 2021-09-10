# Yorc Changelog

## UNRELEASED

### FEATURES

* Allow to replay workflow steps even if they are not in error ([GH-771](https://github.com/ystia/yorc/issues/771))
* Workflows steps replays on error ([GH-753](https://github.com/ystia/yorc/issues/753))

### ENHANCEMENTS

* Support Alien4Cloud 3.3.0 ([GH-773](https://github.com/ystia/yorc/issues/773))
* Slurm: Use sacct to retrieve job status when scontrol show job does not show the job anymore ([GH-757](https://github.com/ystia/yorc/issues/757))
* Add basic support for ssh on Windows ([GH-751](https://github.com/ystia/yorc/issues/751))
* Add the ability to define OpenStack Compute Instance user_data ([GH-735](https://github.com/ystia/yorc/issues/735))

### BUG FIXES

* [Bootstrap] Error in AWS location configuration for the bootstrapped Yorc ([GH-762](https://github.com/ystia/yorc/issues/762))
* Over-consumption of Consul connections ([GH-745](https://github.com/ystia/yorc/issues/745))
* Yorc panics attempting to print an error handling a script execution stdout ([GH-741](https://github.com/ystia/yorc/issues/741))
* Error submitting a SLURM job with no execution option ([GH-739](https://github.com/ystia/yorc/issues/739))
* Workflow with asynchronous action never stops after another step failure  ([GH-733](https://github.com/ystia/yorc/issues/733))

### ENGINEERING

* Generate a checksum file for release artifacts and sign it ([GH-755](https://github.com/ystia/yorc/issues/755))

## 4.2.0-milestone.1 (May 06, 2021)

### ENHANCEMENTS

* Support Alien4Cloud 3.2.0 ([GH-723](https://github.com/ystia/yorc/issues/723))

### BUG FIXES

* Can't bootstrap Yorc as BinTray is now unavailable ([GH-727](https://github.com/ystia/yorc/issues/727))

## 4.1.0 (April 11, 2021)

### DEPENDENCIES

* The orchestrator requires now at least Ansible 2.10.0 (upgrade from 2.7.9 introduced in [GH-648](https://github.com/ystia/yorc/issues/648))
* The orchestrator requires now at least Terraform OpenStack Provider version 1.32.0 (upgrade from 1.9.0 introduced in [GH-703](https://github.com/ystia/yorc/issues/703))


### FEATURES

* Added a tasks dispatcher metrics refresh time configuration parameter ([GH-715](https://github.com/ystia/yorc/issues/715))
* Added an ElasticSearch store for events and logs ([GH-658](https://github.com/ystia/yorc/issues/658))
* [Slurm] Expose Slurm scontrol show job results as job attributes ([GH-664](https://github.com/ystia/yorc/issues/664))

### SECURITY FIXES

* Fix [vulnerability in golang.org/x/crypto/ssh](https://snyk.io/vuln/SNYK-GOLANG-GOLANGORGXCRYPTOSSH-551923) by upgrading dependency

### ENHANCEMENTS

* Alllow shards and replicas configuration for Elastic storage ([GH-722](https://github.com/ystia/yorc/issues/722))
* Add a new synchronous purge API endpoint ([GH-707](https://github.com/ystia/yorc/issues/707))
* Should be able to specify the type of volume when creating an openstack instance ([GH-703](https://github.com/ystia/yorc/issues/703))
* Support ssh connection retries ([GH-688](https://github.com/ystia/yorc/issues/688))
* Remove useless/cluttering logs ([GH-681](https://github.com/ystia/yorc/issues/681))
* Should be able to specify edcsa or rsa ssh keys in gcloud compute instances metadata ([GH-697](https://github.com/ystia/yorc/issues/697))
* Add the ability to define OpenStack Compute Instance metadata ([GH-687](https://github.com/ystia/yorc/issues/687))
* Support Alien4Cloud 3.0 ([GH-689](https://github.com/ystia/yorc/issues/689))
* Upgrade Ansible version from 2.7.9 to 2.10.0 ([GH-648](https://github.com/ystia/yorc/issues/648))
* Alien4Cloud download URL change ([GH-637](https://github.com/ystia/yorc/issues/637))
* Enhance logs and events long-polling performances on file storage ([GH-654](https://github.com/ystia/yorc/issues/654))

### BUG FIXES

* Yorc panics on ElasticSearch store error ([GH-719](https://github.com/ystia/yorc/issues/719))
* Error when storing runtime attributes of google subnet ([GH-713](https://github.com/ystia/yorc/issues/713))
* Bootstrap fails on hosts where a version of ansible < 2.10.0 is installed ([GH-695](https://github.com/ystia/yorc/issues/695))
* Panic due to nil pointer dereference may happen when retrieving a workflow ([GH-691](https://github.com/ystia/yorc/issues/691))
* Empty directories not removed after ansible executions can lead to inodes exhaustion ([GH-683](https://github.com/ystia/yorc/issues/683))
* Yorc generates forcePurge tasks on list deployments API endpoint ([GH-674](https://github.com/ystia/yorc/issues/674))
* Yorc is getting slow when there is a lot of tasks ([GH-671](https://github.com/ystia/yorc/issues/671))
* Yorc does not build on Go1.15 ([GH-665](https://github.com/ystia/yorc/issues/665))
* Concurrency issue in workflows execution may lead to a task never reaching a terminal status ([GH-659](https://github.com/ystia/yorc/issues/659))
* Bootstrap on Centos 7 on GCP fails ([GH-649](https://github.com/ystia/yorc/issues/649))
* Kubernetes Client uses deprecated apis removed on recent versions of K8S (v1.17+) ([GH-645](https://github.com/ystia/yorc/issues/645))
* Bootstrap may failed with a nil pointer error if download of a component fails ([GH-634](https://github.com/ystia/yorc/issues/634))
* Missing concurrency limit during data migration for logs and events file storage ([GH-640](https://github.com/ystia/yorc/issues/640))
* Unable to undeploy a deployment in progress from Alien4Cloud ([GH-630](https://github.com/ystia/yorc/issues/630))

## 4.0.0 (April 17, 2020)

### BUG FIXES

* Location is not listed if its properties are missing ([GH-625](https://github.com/ystia/yorc/issues/625))
* Sometimes Yorc bootstrap fails to start local Yorc instance because Consul is not properly started ([GH-623](https://github.com/ystia/yorc/issues/623))

## 4.0.0-rc.1 (March 30, 2020)

### FEATURES

* Support TOSCA 1.3 inputs/outputs for workflows ([GH-556](https://github.com/ystia/yorc/issues/556))

### ENHANCEMENTS

* Yorc Tasks should have an error message when appropriate ([GH-613](https://github.com/ystia/yorc/issues/613))

### BUG FIXES

* CLI yorc locations list doesn't return HostsPool information ([GH-615](https://github.com/ystia/yorc/issues/615))
* Check for location type mismatch ([GH-550](https://github.com/ystia/yorc/issues/550))

## 4.0.0-M10 (March 10, 2020)

### FEATURES

* Add a consumable resources concept to the Hosts Pool ([GH-205](https://github.com/ystia/yorc/issues/205))

### BUG FIXES

* Yorc Bootstrap doesn't uninstall yorc binary ([GH-605](https://github.com/ystia/yorc/issues/605))
* Document server info REST API endpoint and change returned JSON to comply to our standards ([GH-609](https://github.com/ystia/yorc/issues/609))
* Default cache size for file storage is too large ([GH-612](https://github.com/ystia/yorc/issues/612))

## 4.0.0-M9 (February 14, 2020)

### FEATURES

* Host pool election for compute allocation can be more relevant ([GH-83](https://github.com/ystia/yorc/issues/83))

### BUG FIXES

* Tosca public_ip_address attribute is wrongly set with private address for hosts pool computes ([GH-593](https://github.com/ystia/yorc/issues/593))
* REQ_TARGET keyword on TOSCA doesn't work with requirement type ([GH-598](https://github.com/ystia/yorc/issues/598))
* Failure running workflow with inline activity ([GH-592](https://github.com/ystia/yorc/issues/592))

## 4.0.0-M8 (January 24, 2020)

### BREAKING CHANGES

#### Refactor Yorc storage system

Yorc supports several store types and store implementations that can be configured by users correspondingly with their needs.
See below enhancement allowing to implement a File+Cache sorage backend.

#### Changes in Yorc metric namespace

In order to improve the observability of Yorc execution, the exposed metrics' names were modified.
Now labels are used which allow to provide metric trees ([GH-297](https://github.com/ystia/yorc/issues/297)).

#### Changes on the deployments API

As deployments are from now stored by JSON, some functions have been changed:

```
func GetOperationPathAndPrimaryImplementation(ctx context.Context, deploymentID, nodeTemplateImpl, nodeTypeImpl, operationName string) (string, string, error)
func GetRequirementsKeysByTypeForNode(ctx context.Context, deploymentID, nodeName, requirementType string) ([]string, error)
func GetRequirementKeyByNameForNode(ctx context.Context, deploymentID, nodeName, requirementName string) (string, error)
func ReadWorkflow(kv *api.KV, deploymentID, workflowName string) (tosca.Workflow, error)
```

And substituted by:

```
func GetOperationImplementation(ctx context.Context, ...) (*tosca.Implementation, error)
func GetRequirementsByTypeForNode(ctx context.Context, deploymentID, nodeName, requirementType string) ([]Requirement, error)
func GetWorkflow(ctx context.Context, deploymentID, workflowName string) (*tosca.Workflow, error)
```

### FEATURES

* Volume attachment on AWS Compute nodes ([GH-122](https://github.com/ystia/yorc/issues/122))

### ENHANCEMENTS

* Should be able to bootstrap Yorc on OpenStack with Identity API v3 ([GH-575](https://github.com/ystia/yorc/issues/575))
* Refactor deployments package to be able to use different storage backends - part Two: Consul as default Deployments store implementation ([GH-530](https://github.com/ystia/yorc/issues/530))
* Implement a File+Cache storage backend for static parts of deployments (GH-554](https://github.com/ystia/yorc/issues/554))
* Refactor logs to allow to config new implementations (GH-552](https://github.com/ystia/yorc/issues/552))
* Refactor deployments events to allow to use different backends (GH-553](https://github.com/ystia/yorc/issues/553))

### BUG FIXES

* Deployment stuck and cannot be resumed in certain circumstances ([GH-563](https://github.com/ystia/yorc/issues/563))
* Yorc bootstrap on 4.0.0-M7 doesn't work unless an alternative download URL is provided for Yorc ([GH-561](https://github.com/ystia/yorc/issues/561))
* Location properties stored in Vault are no longer resolvable ([GH-565](https://github.com/ystia/yorc/issues/565))
* If no locations configured when first starting a Yorc server, the command "locations apply config_locations_file_path" won't work ([GH-574](https://github.com/ystia/yorc/issues/5674))
* An error during deployment purge may let the deployment in a wrong state ([GH-572](https://github.com/ystia/yorc/issues/572))
* Can have current deployment and undeployment on the same application on specific conditions ([GH-567](https://github.com/ystia/yorc/issues/567))
* API calls to deploy and update a deployment will now prevent other API calls that may modify a deployment to run at the same time
* Yorc HTTP health check is defined on localhost address ([GH-585](https://github.com/ystia/yorc/issues/585))
* Unable to get network ID attribute for Openstack many compute instances ([GH-584](https://github.com/ystia/yorc/issues/584))

## 4.0.0-M7 (November 29, 2019)

### BREAKING CHANGES

#### Changes in the REST API

##### Deployment updates

Until now deployments updates (which is a premium feature) were made on the same API operation than submitting a deployment with a given identifier:
`PUT /deployments/<deployment_id>`.
The decision on updating a deployment or creating a new one was based on the existence or not of a deployment with the given identifier.

This was confusing and makes error handling on clients difficult. So we decided to split those operations on two different API endpoints.
Submitting a deployment with a given identifier remain as it was, but updating a deployment is now available on the `PATCH /deployments/<deployment_id>` endpoint.

Trying to update a deployment on an OSS version will now result in a `401 Forbiden` error instead of a `409 Conflict` error previously.
Submitting a deployment with an identifier that is already used will still result in a `409 Conflict` error but without an informative message indicating that a
premium version is required to perform a deployment update.

This is tracked on ([GH-547: Refactor Deployment updates API](https://github.com/ystia/yorc/issues/547)).

#### Others

* API /health changed to /server/health ([GH-551](https://github.com/ystia/yorc/issues/551))

### FEATURES

* Add support for using bastion hosts to provision instances with ansible and terraform ([GH-128](https://github.com/ystia/yorc/issues/128))
* Enrich Yorc REST API with endpoints and handlers for locations management ([GH-479](https://github.com/ystia/yorc/issues/479))
* Loading bar while bootstrap ([GH-254](https://github.com/ystia/yorc/issues/254))

### ENHANCEMENTS

* Allow to specify query parameters in infrastructure usage queries ([GH-543](https://github.com/ystia/yorc/issues/543))

### BUG FIXES

* Duplicate SLURM job info log in case of failure ([GH-545](https://github.com/ystia/yorc/issues/545))

## 4.0.0-M6 (November 08, 2019)

### ENHANCEMENTS

* Add Consul DB migration of hostspools due to locations modifications ([GH-531](https://github.com/ystia/yorc/issues/531))
* Allow to provide inline configuration options for generated sbatch scripts ([GH-537](https://github.com/ystia/yorc/issues/537))
* Optionally source an environment file before submitting slurm jobs ([GH-541](https://github.com/ystia/yorc/issues/541))
* Update Infrastructure collector feature to handle locations ([GH-533](https://github.com/ystia/yorc/issues/533))
* Refactor deployments package to be able to use different storage backends - part One: Reduce Consul coupling / Add Context parameter ([GH-530](https://github.com/ystia/yorc/issues/530))

### BUG FIXES

* YORC_LOCATIONS_FILE_PATH env variable and locations_file_path command flag are not working ([GH-535](https://github.com/ystia/yorc/issues/535))

## 4.0.0-M5 (October 11, 2019)

### BREAKING CHANGES

The support of locations in Yorc (issue [GH-478](https://github.com/ystia/yorc/issues/478)), provides the ability to create several locations of a given infrastructure type (Openstack, Google Cloud, AWS, SLURM, Hosts Pool).

It introduces breaking changes in Yorc configuration as described below.

It is not anymore possible to define infrastructure properties through:

* environment variables, like `YORC_INFRA_OPENSTACK_AUTH_URL`
* yorc server command flags, like `--infrastructure_openstack_auth_url`.

Yorc configuration file does not anymore contain infrastructures definitions. This configuration provides now the path to a file defining locations for any of these types: Openstack, Google Cloud, AWS, SLURM.

For example, this Yorc configuration file in previous version allowed to define a single infrastructure of type OpenStack:

```yaml
resources_prefix: yorc-
consul:
  address: http://consul-host:8500
  datacenter: dc1
infrastructures:
  openstack:
    auth_url: http://1.2.3.4:5000
    tenant_name: mytenant
    user_name: myuser
    password: mypasswd
    region: RegionOne
    private_network_name: private-net
    default_security_groups: [group1,default]
```

This becomes now in this version:

```yaml
resources_prefix: yorc-
consul:
  address: 127.0.0.1:8500
locations_file_path: /path/to/locations.yaml
```

And file `/path/to/locations.yaml` contains locations definitions, for example here two OpenStack locations :

```yaml
locations:
  - name: myFirstOpenStackLocation
    type: openstack
    properties:
      auth_url: http://1.2.3.4:5000
      tenant_name: mytenant
      user_name: myuser
      password: mypasswd
      region: RegionOne
      private_network_name: private-net
      default_security_groups: [group1,default]
  - name: mySecondOpenStackLocation
    type: openstack
    properties:
      auth_url: http://5.6.7.8:5000
      tenant_name: mytenant
      user_name: myuser
      password: mypasswd
      region: RegionOne
      private_network_name: private-net
      default_security_groups: [group1,default]
```

When a Yorc server is starting and has no location defined yet, it will read this locations configuration file and create the corresponding locations.
Once this has been done, the locations configuration file won't be used anymore. Next version of Yorc will provide CLI and APIs allowing to create/update/delete locations.

Regardings Hosts Pool, the file format allowing to define one Hosts Pool hasn't changed.
But the CLI commands have now a mandatory argument to provide the location name:  `-l locationName` or `--location locationName`.

For example, this command was executed in previous Yorc version to create/update a Hosts Pool from a file:

```bash
yorc hp apply myhostspool.yaml
```

It is now:

```bash
yorc hp apply -l myLocation myhostspool.yaml
```

See Yorc documentation for additional details:

* section [configuration](https://yorc.readthedocs.io/en/latest/configuration.html)
* section on [CLI commands related to Hosts pool](https://yorc.readthedocs.io/en/latest/cli.html#cli-commands-related-to-hosts-pool)

### FEATURES

* Yorc support of Kubernetes StatefulSet ([GH-206](https://github.com/ystia/yorc/issues/206))
* Add support for asynchronous operations execution on plugins ([GH-525](https://github.com/ystia/yorc/issues/525))

### ENHANCEMENTS

* Locations concept in Yorc ([GH-478](https://github.com/ystia/yorc/issues/478))

### BUG FIXES

* Kubernetes Jobs do not support Service IP injection ([GH-528](https://github.com/ystia/yorc/issues/528))
* BadAccess error may be thrown when trying to resolve a TOSCA function end up in error ([GH-526](https://github.com/ystia/yorc/issues/526))
* Fix possible overlap on generated batch wrappers scripts when submitting several singularity jobs in parallel ([GH-522](https://github.com/ystia/yorc/issues/522))
* Bootstrap wrongly configures on-demand resources on OpenStack ([GH-520](https://github.com/ystia/yorc/issues/520))

## 4.0.0-M4 (September 19, 2019)

### BUG FIXES

* Fixed a bug preventing OpenStack Networks from being created ([GH-515](https://github.com/ystia/yorc/issues/515))
* Having a deployment named as a prefix of another one causes several issues ([GH-512](https://github.com/ystia/yorc/issues/512))
* A deployment may disappear from the deployments list while its currently running a purge task ([GH-504](https://github.com/ystia/yorc/issues/504))
* A4C Logs are displaying stack error when workflow step fails ([GH-503](https://github.com/ystia/yorc/issues/503))
* Bootstrap on OpenStack doesn't allow floating IP provisioning ([GH-516](https://github.com/ystia/yorc/issues/516))

## 4.0.0-M3 (August 30, 2019)

### SECURITY FIXES

* Fixed a vulnerability that could allow an attacker to read the content of a file stored on Yorc host by pretending it is an ssh key. ([GHSA-8vhw-qv5r-38h5](https://github.com/ystia/yorc/security/advisories/GHSA-8vhw-qv5r-38h5))

### BUG FIXES

* Undeploying an application with a running workflow should not be possible ([GH-460](https://github.com/ystia/yorc/issues/460))

## 4.0.0-M2 (August 09, 2019)

### ENHANCEMENTS

* Should support the creation of OpenStack Compute instances using bootable volume ([GH-461](https://github.com/ystia/yorc/issues/461))
* Allow to disable automatic Consul snapshots and restore when upgrading Yorc using ̀`YORC_DISABLE_CONSUL_SNAPSHOTS_ON_UPGRADE` env variable ([GH-486](https://github.com/ystia/yorc/issues/486))
* Allow to update instance attribute when creating attribute notifications ([GH-491](https://github.com/ystia/yorc/issues/491))
* Use the new Yorc plugin provided by Alien4Cloud (the Yorc Provider) in the bootstrap process. ([GH-494](https://github.com/ystia/yorc/issues/494))
* Missing documentation on premium features ([GH-407](https://github.com/ystia/yorc/issues/407))

## 4.0.0-M1 (July 12, 2019)

### BREAKING CHANGES

* Start v4.0 cycle ([GH-444](https://github.com/ystia/yorc/issues/444)):
  * deprecated API functions are now removed
  * the former and deprecated way to handle Kubernetes deployments is not supported anymore

### BUG FIXES

* Failure to deploy/undeploy big application: Transaction contains too many operations ([GH-484](https://github.com/ystia/yorc/issues/484))
* Wrong resources allocation on shareable Hosts Pool ([GH-426](https://github.com/ystia/yorc/issues/426))
* Deleting one host in Pool deletes other hosts having as prefix the deleted hostname ([GH-430](https://github.com/ystia/yorc/issues/430))
* Yorc should support long standard operation names as well as short ones ([GH-300](https://github.com/ystia/yorc/issues/300))
* Fix attributes notifications for services (substitutions) ([GH-423](https://github.com/ystia/yorc/issues/423))
* Monitoring can be stopped before the job termination ([GH-438](https://github.com/ystia/yorc/issues/438))
* mem_per_node slurm option parameter is limited to integer number of GB ([GH-446](https://github.com/ystia/yorc/issues/446))
* Job node state remains to "executing" when Ansible job fails ([GH-455](https://github.com/ystia/yorc/issues/455))
* Panic occurs uploading job slurm artifacts during load test ([GH-465](https://github.com/ystia/yorc/issues/465))

### ENHANCEMENTS

* Support OpenStack Block storage API v3 ([GH-440](https://github.com/ystia/yorc/issues/440))
* Expose bypass error parameter on workflow ([GH-425](https://github.com/ystia/yorc/issues/425))
* Support Alien4Cloud 2.2 ([GH-441](https://github.com/ystia/yorc/issues/441))
* Allow to provide extra env vars to Alien4Cloud during bootstrap ([GH-452](https://github.com/ystia/yorc/issues/452))
* Port CLI and Rest API minor changes for premium update feature ([GH-467](https://github.com/ystia/yorc/issues/467))
* Port changes for update nodes ([GH-476](https://github.com/ystia/yorc/issues/476))

## 3.2.0 (May 31, 2019)

### ENHANCEMENTS

* Bootstrap support of Ubuntu 1904 ([GH-419](https://github.com/ystia/yorc/issues/419))
* Print plugin logs in higher level than DEBUG ([GH-329](https://github.com/ystia/yorc/issues/329))
* Slurm job logs are displayed many time ([GH-397](https://github.com/ystia/yorc/issues/397))
* Allow to configure resources prefix for bootstrapped Yorc ([GH-399](https://github.com/ystia/yorc/issues/399))

### BUG FIXES

* Undeployment of a Hosts Pool bootstrapped setup doesn't clean correctly Hosts Pool ([GH-406](https://github.com/ystia/yorc/issues/406))
* Failure to deploy an application with Compute resources constraints on Hosts Pool ([GH-409](https://github.com/ystia/yorc/issues/409))
* Yorc server crashes on error: fatal error: concurrent map write ([GH-413](https://github.com/ystia/yorc/issues/413))
* Yorc bootstrap on HostPool in interactive mode should ask for hosts private IP addresses also ([GH-411](https://github.com/ystia/yorc/issues/411))
* Secure bootstrap on Hosts Pool fails configuring infra, error "playbooks must be a list of plays" ([GH-396](https://github.com/ystia/yorc/issues/396))
* Kubernetes infrastructure configuration support in Yorc is not able to detect erroneous config file path ([GH-378](https://github.com/ystia/yorc/issues/378))
* Emit a persistent event on deployment purge ([GH-402](https://github.com/ystia/yorc/issues/402))
* Invalid memory address panic appends on a workflow with non-existent on success step reference ([GH-417](https://github.com/ystia/yorc/issues/417))
* Erroneous Warning message on purged deployments timestamp ([GH-421](https://github.com/ystia/yorc/issues/421))

## 3.2.0-RC1 (May 10, 2019)

### BUG FIXES

* Can't deploy applications using compute instances filters on Hosts Pools ([GH-385](https://github.com/ystia/yorc/issues/385))
* Unexpected deployment deletion during topology unmarshalling ([GH-375](https://github.com/ystia/yorc/issues/375))
* Parsing of a description field of an TOSCA interface is interpreted as an operation ([GH-372](https://github.com/ystia/yorc/issues/372))
* Yorc does not support python3 ([GH-319](https://github.com/ystia/yorc/issues/319))
* Migrate from Oracle JDK to OpenJDK when bootstrapping Yorc/Alien4Cloud ([GH-383](https://github.com/ystia/yorc/issues/383))
* Monitoring causes Yorc and Consul CPU over consumption ([GH-388](https://github.com/ystia/yorc/issues/388))
* Bootstrap step YorcOnDemandLocationResources fails on setup with http proxy defined ([GH-384](https://github.com/ystia/yorc/issues/384))

### ENHANCEMENTS

* Add a note on ansible upgrade in documentation ([GH-373](https://github.com/ystia/yorc/issues/373))
* Policies API update ([GH-380](https://github.com/ystia/yorc/issues/380))
* Reduce the volume of data stored in Consul by removing builtin types duplicates on deployments ([GH-371](https://github.com/ystia/yorc/issues/371))
* Execute custom workflows and custom commands on selected instances ([GH-107](https://github.com/ystia/yorc/issues/107))
* Support OpenStack authentication with user domain ([GH-355](https://github.com/ystia/yorc/issues/355))

## 3.2.0-M5 (April 19, 2019)

### FEATURES

* Implement an anti-affinity placement policy for Openstack ([GH-84](https://github.com/ystia/yorc/issues/84))
* Allow to configure Ansible configuration file ([GH-346](https://github.com/ystia/yorc/issues/346))
* Monitor deployed services liveness ([GH-104](https://github.com/ystia/yorc/issues/104))

### ENHANCEMENTS

* Add job status feedback for slurm and k8s jobs ([GH-351](https://github.com/ystia/yorc/issues/351))
* Upgrade Ansible to 2.7.9 ([GH-364](https://github.com/ystia/yorc/issues/364))
* Reduce the volume of data stored in Consul part 1 ([GH-361](https://github.com/ystia/yorc/issues/361))

### BUG FIXES

* Unable to delete a deployment with non-conform topology ([GH-368](https://github.com/ystia/yorc/issues/368))

## 3.2.0-M4 (March 29, 2019)

### ENHANCEMENTS

* REST API doc changes on deployment update support in premium version ([GH-352](https://github.com/ystia/yorc/issues/352))
* Bootstrap Yorc with a Vault instance ([GH-282](https://github.com/ystia/yorc/issues/282))
* Refactor Slurm jobs ([GH-220](https://github.com/ystia/yorc/issues/220))
* Yorc does not log a slurm command error message, making diagnostic difficult ([GH-348](https://github.com/ystia/yorc/issues/348))

### FEATURES

* Yorc hostspool now allows more filtering ([GH-89](https://github.com/ystia/yorc/issues/89))
* Yorc support of kubernetes PersistentVolumeClaim ([GH-209](https://github.com/ystia/yorc/issues/209))

### BUG FIXES

* Bootstrap using a premium version of Alien4Cloud fails to configure https/SSL ([GH-345](https://github.com/ystia/yorc/issues/345))
* Deployment fails on error "socket: too many open files" ([GH-334](https://github.com/ystia/yorc/issues/334))
* Yorc bootstrap does not correctly treat default alien4cloud version download ([GH-286](https://github.com/ystia/yorc/issues/286))
* Attribute notification is not correctly set with HOST keyword ([GH-338](https://github.com/ystia/yorc/issues/338))
* Custom command events doesn't provide enough information ([GH-324](https://github.com/ystia/yorc/issues/324))
* Update Default Oracle JDK download URL as the previous one is not available for download anymore ([GH-341](https://github.com/ystia/yorc/issues/341))
* Bad notifications storage with several notified for one attribute notifier ([GH-343](https://github.com/ystia/yorc/issues/343))

## 3.2.0-M3 (March 11, 2019)

### BUG FIXES

* Yorc panics on segmentation violation attempting to deploy Ystia Forge Slurm topology ([GH-321](https://github.com/ystia/yorc/issues/321))
* Panic can append when undeploying Slurm computes ([GH-326](https://github.com/ystia/yorc/issues/326))

### FEATURES

* Yorc supports Slurm Accounting ([GH-280](https://github.com/ystia/yorc/issues/280))
* Yorc supports Slurm reservation ([GH-132](https://github.com/ystia/yorc/issues/132))

## 3.2.0-M2 (February 15, 2019)

### DEPENDENCIES

* Technical update to use Alien4Cloud 2.1.1 (Used in bootstrap)

### BUG FIXES

* Purging n deployment in parallel, one can fail on error: Missing targetId for task with id ([GH-293](https://github.com/ystia/yorc/issues/293))
* Deployment with a topology parsing error remains in initial status ([GH-283](https://github.com/ystia/yorc/issues/283))
* Interface name is not retrieved from custom command Rest request ([GH-287](https://github.com/ystia/yorc/issues/287))
* Instances are adding into topology before creating task ([GH-289](https://github.com/ystia/yorc/issues/289))
* Missing events for uninstall workflow in purge task ([GH-302](https://github.com/ystia/yorc/issues/302))
* All ssh connections to Slurm are killed if ssh server has reached the max number of allowed sessions ([GH-291](https://github.com/ystia/yorc/issues/291))
* It can take a considerable delay for a deployment to change status to UNDEPLOYMENT_IN_PROGRESS ([GH-306](https://github.com/ystia/yorc/issues/306))
* Slurm job monitoring is not designed for concurrency ([GH-308](https://github.com/ystia/yorc/issues/308))
* SSH Session pool: Panic if connection failed, this impacts Slurm infrastructure ([GH-315](https://github.com/ystia/yorc/issues/315))

### ENHANCEMENTS

* Bootstrap a secure Yorc setup ([GH-179](https://github.com/ystia/yorc/issues/179))
* Yorc bootstrap should save input values used to bootstrap a setup ([GH-248](https://github.com/ystia/yorc/issues/248))
* Publish value change event for instance attributes ([GH-222](https://github.com/ystia/yorc/issues/222))
* Move to Go modules to manage dependencies ([GH-183](https://github.com/ystia/yorc/issues/183))
* Document How to create a Yorc Plugin ([GH-119](https://github.com/ystia/yorc/issues/119))
* Slurm user credentials can be defined as slurm deployment topology properties, as an alternative to yorc configuration properties ([GH-281](https://github.com/ystia/yorc/issues/281))

## 3.2.0-M1 (January 28, 2019)

### BUG FIXES

* Can't deploy applications using a secured yorc/consul ([GH-274](https://github.com/ystia/yorc/issues/274))
* K8S jobs namespace should not be removed if its provided ([GH-245](https://github.com/ystia/yorc/issues/245))
* Unable to purge an application that appears in the list ([GH-238](https://github.com/ystia/yorc/issues/238))

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
