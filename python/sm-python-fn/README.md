# Function Title

## Description

This function helps users <do what>. Please go through the steps below to deploy, customize and build this function. 

### Deploy function (without modification)
Step 1 - Clone repo

```
git clone <git-clone-url>
cd <function-folder-path>
```

Step 2 - View the `stack.yml` and modify file as required

```
version: 1.0
provider:
  name: openfaas                                  
  gateway: <http(s)://VEBA_FQDN>        # OpenFaaS URL (VEBA endpoint) 
functions:
  <sm-fn-name>:                         # name of the function (unique in your k8s cluster)
    lang: python
    handler: ./handler                  # folder name which has your scripts
    image: <docker/sm-veba-fn>:latest   # docker container to pull for this deployment
    environment:
      write_debug: true
      read_debug: true
    annotations:
      topic: "VmPoweredOffEvent"         # comma separated events that trigger this fn
```

Step 3 - Login to the OpenFaaS gateway on vCenter Event Broker Appliance

```
export OPENFAAS_URL=https://veba-fqdn

faas-cli login --username admin --password-stdin --tls-no-verify
```

Step 4 - Deploy function to vCenter Event Broker Appliance

```
faas-cli deploy -f stack.yml --tls-no-verify
```

Step 5 - Tail the logs of the <sm-fn-name> function

```
faas-cli logs <sm-fn-name> --tls-no-verify
```

Step 6 - Trigger the vCenter Event and you should see output like the following in the console:

```
show your function output here
.....
....
..
.
```

### Troubleshooting the function

If you function is not working as intended, tail the logs of the <sm-fn-name> function using faas-cli as show below. 

```
faas-cli logs <sm-fn-name> --tls-no-verify
```