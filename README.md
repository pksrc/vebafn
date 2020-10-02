# vebafn
A collection of templates and demo apps for [Vmware Event Broker Appliance](https://vmweventbroker.io)

## starter-templates: 
A set of quick starter template for writing functions for VEBA w/ best practice considerations

```
git clone https://github.com/pksrc/vebafn
cd vebafn/starter-templates/  
git checkout master
```

## vmworld2020: 
A starter template for those watching VMworld 2020 Session [Arm Yourself w/Event-Driven Functions & Reimagine SDDC Capabilities - HCP1404](https://my.vmworld.com/widget/vmware/vmworld2020/catalog/session/15863800295950014HrA) and wanting to follow along, you can find the go and python set of files that we started with during our joint coding sessions. 

```
git clone https://github.com/pksrc/vebafn
cd vebafn/vmworld2020/ 
git checkout master
```

## vm-self-service-app
All the functions (to be deployed with [VEBA](https://vmweventbroker.io) deployed with [OpenFaaS](https://www.openfaas.com)) that make up the VM self-service app 
 - User Driven through Slack bot (Python based Gateway, PowerCLI based VM Lifecycle function)
   - Create a VM
   - Clone from a VM
   - Clone from a Template
   - PowerCycle a VM
   - Change the Spec of a VM
   - Run a PowerCLI command on vCenter from Slack (very dangerous)
 - Event Driven through vCenter Alarms (Go based Crisis Remediation functions)
   - Vertical Scaling a VM / Desired State Management of a VM 
   - Auto Storage DRS

```
git clone https://github.com/pksrc/vebafn
cd vebafn/vm-self-service-app
git checkout master
```
