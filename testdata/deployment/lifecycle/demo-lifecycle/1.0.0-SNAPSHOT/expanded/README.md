A topology to illustrate the behavior of nodes and their operations during workflow lifecycle.

Nodes and relationship operation are logged in a central registry and are viewable using a web browser. You can also explore the environment variables for each operation.

![Topology](https://raw.githubusercontent.com/alien4cloud/samples/master/demo-lifecycle/img/lifecycle.png)

The topology contains:

- A ComputeRegistry that host a Apache + PHP + few PHP scripts that acts as a **Registry** for others nodes.
- 2 other computes (ComputeA and ComputeB) host:
   - a **RegistryConfigurer** that is linked to the Registry (actually just peek it's IP and put it in /etc/hosts)
   - A couple of **GenericHost** + **Generic**

About components:

- **Generic** is hosted on **GenericHost**
- **GenericHost** as input on *configure* operation
- The 2 **Generic**s are connected together (with inputs on *add_source* and *add_target* operations)

Once deployed, just follow the link given by the topology output property Registry.url  

![Topology](https://raw.githubusercontent.com/alien4cloud/samples/master/demo-lifecycle/img/lifecycle-webpage.png)

Then :

- Click on **Logs** to vizualise the operations for this instance.

```
#18 - create
#40 - pre_configure_source/GenericHostA/0/GenericHostA_67d5e
#42 - pre_configure_source/GenericB/0/GenericB_5dd0c
#43 - configure
#45 - post_configure_source/GenericB/0/GenericB_5dd0c
#46 - post_configure_source/GenericHostA/0/GenericHostA_67d5e
#50 - start
#58 - add_target/GenericB/1/GenericB_1082d
#60 - add_target/GenericHostA/0/GenericHostA_67d5e
#62 - add_target/GenericB/0/GenericB_5dd0c
#101 - add_target/GenericB/2/GenericB_8f14b
```
Each line is an operation call with tier information when related to a relationship.

- Click on **env logs** to vizualise the environment variables for a given operation.

```
CLOUDIFY_DAEMON_NAME=ComputeA_46ba1
CELERY_WORK_DIR=/home/ubuntu/ComputeA_46ba1/work
XDG_SESSION_ID=4
MANAGER_FILE_SERVER_BLUEPRINTS_ROOT_URL=http://172.31.25.47:53229/blueprints
NODE=GenericHostA
HOST=ComputeA
SHELL=/bin/bash
TERM=vt100
AWS_CONFIG_PATH=/etc/cloudify/aws_plugin/boto
CELERY_TASK_SERIALIZER=json
GenericHostA_67d5e_IP_ADDR=172.31.43.240
USER=ubuntu
IP_ADDR=172.31.39.95
INSTANCES=GenericHostA_1a3c7,GenericHostA_67d5e
CLOUDIFY_DAEMON_USER=ubuntu
GenericHostA_1a3c7_IP_ADDR=172.31.39.95
MANAGER_FILE_SERVER_URL=http://172.31.25.47:53229
CELERY_RESULT_SERIALIZER=json
...
```

## Requirements

- apt-get
- wget --no-proxy

## TODOs

- [ ] apache port attribute should be used by RegistryConfigurer, and Generic & GenericHost
- [ ] simple python scripts could be lighter than apache + php (but for the moment I am not able to get the request body when using HTTPServer and CGIHTTPRequestHandler)
- [ ] apache doc root should be used by Registry
