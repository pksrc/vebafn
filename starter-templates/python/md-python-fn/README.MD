# Function Title

## Description

This function helps users <do what>. Please go through the steps below to deploy, customize and build this function. 

## Deploy function (without modification)

Step 1 - Clone repo

```bash
git clone git@github.com:pksrc/vebafn.git vebafn
cd vebafn/python/md-python-fn
```

Step 2 - View the `stack.yml` and modify file as required

`stack.yml` file is the main descriptor that contains the function-specific settings. Open and edit the `stack.yml`. 

```yaml
version: 1.0
provider:
  name: openfaas                                  
  gateway: <http(s)://VEBA_FQDN>        # OpenFaaS URL (VEBA endpoint) 
functions:
  <md-fn-name>:                         # name of the function (unique in your k8s cluster)
    lang: python
    handler: ./handler                  # folder name which has your scripts
    image: <docker/md-veba-fn>:latest   # docker container to pull for this deployment
    environment:
      write_debug: true
      read_debug: true
    secrets:
      - config                          # configuration for the function
    annotations:
      topic: "VmPoweredOffEvent"        # comma separated events that trigger this fn
```

Step 3 - View and edit the `config.json` 

We will use the [Kubernetes secrets](https://kubernetes.io/docs/concepts/configuration/secret/){:target="_blank"} to hold <what information?>. This secret will be mounted into the function during runtime automatically by the appliance. We only have to create the secret with a simple command through `faas-cli`.

```json
{
    "routing_key": "<replace with your routing key>",
    "event_action": "trigger"
}
```

Step 4 - Login to the OpenFaaS gateway on vCenter Event Broker Appliance and create the secret (the configuration)

```bash
export OPENFAAS_URL=https://veba-fqdn

faas-cli login --username admin --password-stdin --tls-no-verify

# now create the secret
faas-cli secret create config --from-file=config.json --tls-no-verify
```

Step 5 - Deploy function to vCenter Event Broker Appliance

```bash
faas-cli deploy -f stack.yml --tls-no-verify
```

Step 6 - Tail the logs of the <md-fn-name> function

```bash 
faas-cli logs <md-fn-name> --tls-no-verify
```

Step 7 - Trigger the vCenter Event and you should see output like the following in the console:

```bash
#show your function output here
```

## Customize the function

Step 1 - Change the configuration file [config.json](config.json) in this folder:

```json
{
    "routing_key": "<replace with your routing key>",
    "event_action": "trigger" 
}
```

Step 2 - Go ahead and store this configuration file as secret in the appliance.

```bash
# set up faas-cli for first use
export OPENFAAS_URL=https://VEBA_FQDN_OR_IP

faas-cli login -p VEBA_OPENFAAS_PASSWORD --tls-no-verify 

# now create the secret
faas-cli secret create config --from-file=config.json --tls-no-verify
```

> **Note:** Delete the local `config.json` after you're done with this exercise to not expose this sensitive information.

Step 3 - Edit the `stack.yml`

If you are looking to customize the function, you are possibly looking to have the function be triggered for a different `topic`

> **Note:** A key-value annotation under `topic` defines which VM event should trigger the function. A list of VM events from vCenter can be found [here](https://code.vmware.com/doc/preview?id=4206#/doc/vim.event.VmEvent.html). Multiple topics can be specified using a `","` delimiter syntax, e.g. "`topic: "VmPoweredOnEvent,VmPoweredOffEvent"`".

Step 4 - Edit the `handler.py` and `requirements.txt`

Not all events are made the same and thus if you want the function to be invoked for a different event, you'll most likely need to modify the handler.py. The file has sufficient call outs that will guide you through the process. 

If you have any library that is needed, add those to the requirements.txt file. 

### Deploy the function

After you've performed the steps and modifications above, you can go ahead and deploy the function:

```bash
faas-cli template pull # only required during the first deployment
faas-cli up --tls-no-verify
Deployed. 202 Accepted.
```

### Trigger the function

<Do something>, to trigger the function via a `(DRS)VmPoweredOnEvent`. You should see <what is the function expected to do>

## Troubleshooting the function

If you function is not working as intended, tail the logs of the <md-fn-name> function using faas-cli as show below. 

```
faas-cli logs <md-fn-name> --tls-no-verify
```

If your function is not getting invoked, verify:

- The configuration within the `config.json`
- Whether the components can talk to each other (VMware Event Router to vCenter and OpenFaaS, VMware Event Broker Appliance to PagerDuty)
- If you have changed the `topic` in `stack.yml`, please ensure that the Function is also updated to handle the expected event data. 
- Check the logs:

```bash
# Successful log message in the OpenFaaS function
2019/01/25 23:48:55 Forking fprocess.
2019/01/25 23:48:55 Query
2019/01/25 23:48:55 Path  /

{"status": "200", "message": "successfully triggered event action with dedup_key: <redacted>"}
2019/01/25 23:48:56 Duration: 1.551482 seconds
```
