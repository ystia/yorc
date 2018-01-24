# JANUS Orchestrator

[![Build Status](http://129.184.11.224/buildStatus/icon?job=Janus-Engine)](http://129.184.11.224/view/Janus%20Engine/job/Janus-Engine/)

Janus is an hybrid cloud/HPC [TOSCA](http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html) orchestrator.


## How to build Janus

Go 1.9+ is required. The easiest way to install it to use follow [the official guide](https://golang.org/doc/install)

Here is how to install and setup the Janus project:

    sudo apt-get install build-essential git curl
    # Or
    sudo yum install build-essential git curl
    
    # Install GO and set GOPATH
    
    mkdir -p $GOPATH/src/novaforge.bull.com/starlings-janus
    cd $GOPATH/src/novaforge.bull.com/starlings-janus
    git clone ssh://git@novaforge.bull.com:2222/starlings-janus/janus.git
    cd janus

    # Build 
    make tools
    make

## How to test & develop 

Please report to [this cookbook](https://confluence.sdmc.ao-srv.com/x/UoRIAw)
