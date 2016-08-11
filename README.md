# JANUS Orchestrator

Janus is an hybrid cloud/HPC [TOSCA](http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.0/TOSCA-Simple-Profile-YAML-v1.0.html) orchestrator.


## How to build Janus

Go 1.6.2+ is required. The easiest way to install it to use [GVM](https://github.com/moovweb/gvm)

Here is how to install and setup the Janus project:

    sudo apt-get install build-essential git curl
    # Or
    sudo yum install build-essential git curl
    
    bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
    source ~/.bashrc
    gvm install go1.4 -B
    gvm use go1.4
    gvm install go1.6.2 -B -pb -b
    gvm use go1.6.2 --default
    
    # By default GOPATH (where are stored your sources) lives in ~/.gvm/pkgsets/go1.6.2/global
    # You can edit this with 
    gvm pkgenv
    gvm pkgset use global
    
    mkdir -p $GOPATH/src/novaforge.bull.com/starlings-janus
    cd $GOPATH/src/novaforge.bull.com/starlings-janus
    git clone ssh://git@novaforge.bull.com:2222/starlings-janus/janus.git
    cd janus

    go get -v golang.org/x/tools/cmd/stringer

    # Build 
    make
  
##  Run in dev mode

Install GoDep in order to install project dependencies in GOPATH

    go get -v -u github.com/tools/godep
    godep restore -v
    
Build and run consul

    cd ${GOPATH}/src/github.com/hashicorp/consul
    make dev
    ./bin/consul agent -dev -advertise 127.0.0.1
    
Build terraform

    go get -v github.com/hashicorp/terraform
    cd ${GOPATH}/src/github.com/hashicorp/terraform
    make dev
    
Run Janus

    cd $GOPATH/src/novaforge.bull.com/starlings-janus/janus
    make
    ./janus server

For now OpenStack config should be exported before run Janus

     export OS_AUTH_URL=...
     export OS_USERNAME=...
     export OS_PASSWORD=...
     export OS_TENANT_NAME=...
     
    
Deploy a first node
 
    cd $GOPATH/src/novaforge.bull.com/starlings-janus/janus/testdata/deployment
    zip dep.zip dep.yaml
    curl -X POST localhost:8800/deployments -v --data-binary @dep.zip -H "Content-Type: application/zip"

The location header give the URI that should be used to get info on this particular deployment

Check Deployment status

    curl localhost:8800/deployments/<deployment_uuid> -v -H "Accept: application/json"

