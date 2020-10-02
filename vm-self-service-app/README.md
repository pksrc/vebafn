# VM Self Service (demo) App package

## Objective

In the VMworld 2020 session [Arm Yourself w/Event-Driven Functions & Reimagine SDDC Capabilities - HCP1404](https://my.vmworld.com/widget/vmware/vmworld2020/catalog/session/15863800295950014HrA), [PK](https://twitter.com/pkblah) and [Frankie](https://github.com/codegold79) present a demo made up completely of FaaS functions written in Python, Go, and PowerCLI. The purpose of the demo was to show what was possible using [VMware Event Broker Appliance](https://vmweventbroker.io) deployed with [OpenFaaS](https://www.openfaas.com).

The code while completely open sourced and made available here should be used with caution and with complete understanding of how they work. A detailed blog post explaining the functionality will be available [here](https://medium.com/@pkblah). 

> **Obligatory Disclaimer** - This is a working set of functions but DO NOT deploy without proper understanding of the code and expecting support. 

## Functionality

The demo shows scenario where IT can enable Business Units to a VM Self Service capability from Slack (completely powered by VMware Event Broker Appliance). 

1. The first way is user-driven through Slack where one can manage VMs using Slack slash commands.
2. The second way is event-driven, leveraging events from vCenter to attach tags, move a VM from one datastore to another, and reconfigure a VM.

## Components

There are a number of functions that make up the Application - Most are on this [pksrc/vebafn](https://github.com/pksrc/vebafn) repository, and few are on the [VEBA](https://github.com/vmware-samples/vcenter-event-broker-appliance) repository.

> Pre-requisite: VMWare Event Broker Appliance deployed with OpenFaaS ([documentation](https://vmweventbroker.io/kb/install-openfaas))

### Slack Bot (User Driven)

  * IaaS Gateway (written in Python) - [Link](https://github.com/pksrc/vebafn/tree/master/vm-self-service-app/python/iaasgw)
  * VM Lifecycle (written in PowerCLI) - [Link](https://github.com/pksrc/vebafn/tree/master/vm-self-service-app/powercli/vmlifecycle)
  * *Utility Functions* (written in PowerCLI) - [Link](https://github.com/pksrc/vebafn/tree/master/vm-self-service-app/powercli/util)

### Crisis Remediation (Event Driven)

  * Vertical Scaling of a VM (written in Go) 
    * VM Tagging (written in Go and based on [VEBA example go tagging](https://github.com/vmware-samples/vcenter-event-broker-appliance/tree/development/examples/go/tagging)) - [Link](https://github.com/pksrc/vebafn/tree/master/vm-self-service-app/go/go-vm-config-tagger)
    * Reconfigure a VM based on attached tags -  [Link](https://github.com/vmware-samples/vcenter-event-broker-appliance/tree/development/examples/go/vm-reconfig-via-tag)
    * *Utility Function* - Tag generator for configuring VMs [Link](https://github.com/vmware-samples/vcenter-event-broker-appliance/tree/development/examples/go/go-tag-generator)
  * Auto Storage DRS (written in Go) - [Link](https://github.com/vmware-samples/vcenter-event-broker-appliance/tree/development/examples/go/go-vm-datastore-move)


## Deployment

* Update all the Stack.yml with your VEBA_OPENFAAS_ENDPOINT
* Update all the vcconfig.json with the right vCenter Credentials
* You need a public TLS certificate bound to VEBA - [follow guide](https://medium.com/@pkblah/publicly-trusted-tls-for-vmware-eventing-platform-6c6f5d0a14fb)

```zsh
# STEP 1: 
# Download the repository
git clone git@github.com:pksrc/vebafn.git 
cd vebafn
git checkout master

# set up faas-cli for first use
export OPENFAAS_URL=https://pdotk.lab.net
faas-cli login -p VEBA_OPENFAAS_PASSWORD # vCenter Event Broker Appliance is configured with authentication, pass in the password used during the vCenter Event Broker Appliance deployment process

# STEP 2: 
# deploy the gateway
cd vm-self-service-app/python/iaasgw
faas-cli up 

# STEP 3: 
#deploy the vm lifecycle functions
cd ../../powercli/vmlifecycle/ 

faas-cli secret create vcconfigjson --from-file=vcconfig.json # create the secret
faas-cli up

#optional - deploy the utility function
```

```zsh
# STEP 4:
cd ../../go/go-vm-datastore-move
faas-cli secret create vcconfig --from-file=vcconfig.toml
faas-cli template store pull golang-http
faas-cli up -f stack.yml --build-arg GO111MODULE=on

# STEP 5: 
cd ../go-tag-generator
./tag-gen

cd ../go-vm-config-tagger
faas-cli template store pull golang-http
faas-cli up -f stack.yml --build-arg GO111MODULE=on 

git clone git@github.com:vmware-samples/vcenter-event-broker-appliance.git veba
cd veba/examples/go/vm-reconfig-via-tag 
faas-cli template store pull golang-http
faas-cli up -f stack.yml --build-arg GO111MODULE=on
```