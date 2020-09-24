# Demo App package

## Demo Objective

In the VMworld 2020 presentation Arm Yourself with Event-Driven Functions and Reimagine SDDC Capabilities (HCP1404), PK and Frankie present a demo made up of FaaS functions written in Python, Go, and PowerCLI. The purpose of the demo was to show what was possible using event-driven functions.

The code backing the demo is not production ready, but it does give an example of how one might go about making one's own functions.

## What the Demo Does

The demo shows two ways to interact with vSphere via the vCenter Event Broker Appliance (VEBA). The first way is through Slack where one can turn on or off VMs by sending Slack commands.

The second way is by triggering actions on events that come from vCenter. The resulting actions are attaching tags, moving a VM from one datastore to another, and reconfiguring a VM.

## The Functions Components

These are the functions that make up the Demo application. Some are on this pksrc/vebafn repository, and others are on the VEBA repository.

* Slack commands [Link](https://github.com/pksrc/vebafn/tree/master/vmworld2020)
* Attach vm configuration tags on CPU and Memory alarms (based on [go tagging](https://github.com/vmware-samples/vcenter-event-broker-appliance/tree/development/examples/go/tagging) VEBA example) [Link](go-vm-config-tagger)
* Move a VM to another datastore on Datastore alarm [Link](go-vm-datastore-move)
* Reconfigure a VM based on attached tags when it powers off [Link](https://github.com/vmware-samples/vcenter-event-broker-appliance/tree/development/examples/go/vm-reconfig-via-tag)
* Send PagerDuty alert when a VM is Reconfigured [Link](https://github.com/vmware-samples/vcenter-event-broker-appliance/tree/development/examples/go/pagerduty-trigger)

There are also functions that do not make up the demo application, but they help with demo preparation.

* Tag generator for configuring VMs [Link](go-tag-generator)
* Basic outline function [Link](go-outline)
