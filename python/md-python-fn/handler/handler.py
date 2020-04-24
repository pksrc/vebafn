import sys, json, os
import urllib3
import requests
import traceback

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

###
# Any Global variables or  Paths and Endpoints
# EDIT AS REQUIRED
###
API_URL='https://events.pagerduty.com/v2/enqueue'
CONFIG='/var/openfaas/secrets/config'

class FaaSResponse:
    """
    FaaSResponse is a helper class to construct a properly formatted message returned by this function.
    By default, OpenFaaS will marshal this response message as JSON.
    """    
    def __init__(self, status, message):
        """
        https://www.geeksforgeeks.org/__init__-in-python/
        Arguments:
            status {str} -- the response status code
            message {str} -- the response message
        """        
        self.status=status
        self.message=message

class YourClassName:
    """
    YourClassName is the class where you write your core business logic.
    """    

    def __init__(self,conn):
        """
        Arguments:
            conn {session} -- connection to PagerDuty REST API
        """
        ###
        # init method or constructor for YourClassName
        # EDIT AS REQUIRED
        ###
        self.session=conn

    def invoke(self,obj):
        """
        Make a rest api call to Pagerduty Events API
        
        Arguments:
            obj {dict} -- Generated API Body for the PagerDuty Events API

        Returns:
            FaaSResponse -- status code and message
        """
        ###
        # Core logic contained here
        # EDIT AS REQUIRED
        ###
        try:
            resp = self.session.post(API_URL,json=obj)
            resp.raise_for_status()
            resp_body = json.loads(resp.text)
            return FaaSResponse('200', 'Successfully invoked PagerDuty API! dedup_key for this request: {0}'.format(resp_body['dedup_key']))
        except requests.HTTPError as err:
            return FaaSResponse('500', 'Could not invoke PagerDuty API > HTTPError: {0}'.format(err))

def handle(req):
    
    # Validate Event input
    try:
        cevent = json.loads(req)
    except json.JSONDecodeError as err:
        res = FaaSResponse('400','Invalid JSON > JSONDecodeError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return
    
    # Read and Validate Config file
    try: 
        with open(CONFIG, 'r') as configfile:
            config = json.load(configfile)
            
            ###
            # Validating the presence of these keys within the Config JSON
            # EDIT AS REQUIRED
            ###
            routingkey=config['routing_key']
            event_action=config['event_action']
            
    except json.JSONDecodeError as err:
        res = FaaSResponse('400','Invalid JSON > JSONDecodeError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return
    except KeyError as err:
        res = FaaSResponse('400','Required key not found in the configuration > KeyError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return 
    except OSError as err:
        res = FaaSResponse('500','Could not read PagerDuty configuration > OSError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return

    # Assert that the function is able to get the required information from the event and build the request body
    try:
        ###
        # Map the CloudEvent data and build the PagerDuty Event API Request body
        # EDIT AS REQUIRED
        ###
        obj = {
                'payload': {
                    'summary': cevent['data']['FullFormattedMessage'],
                    'timestamp': cevent['data']['CreatedTime'],
                    'source': cevent['source'],
                    'severity': 'info',
                    'component': cevent['data']['Vm']['Name'],
                    'group': cevent['data']['Host']['Name'],
                    'class': cevent['subject']
                },
                'client': 'VMware Event Broker Appliance',
                'client_url': cevent['source'],
                'routing_key': routingkey,
                'event_action': event_action
            }
    except KeyError as err:
        res = FaaSResponse('400','Invalid JSON, required key not found in the provided Event > KeyError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return
    except TypeError as err:
        res = FaaSResponse('400','Invalid JSON, missing required data in the provided Event > TypeError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return

    # Make the Rest Api Call to PagerDuty
    s=requests.Session()
    if(os.getenv("insecure_ssl")):
        s.verify=False
    try:
        ucn = YourClassName(s)
        res = ucn.invoke(obj)
        print(json.dumps(vars(res)))
    except Exception as err:
        res = FaaSResponse('500','Unexpected Error occurred > Exception: {0}'.format(err))
        print(json.dumps(vars(res)))
    
    #Close session
    s.close()

    return

#
## Unit Test - helps with executing the function locally
## Uncomment PDConfig - update the path to the file accordingly
## Uncomment handle('... to test the function with the event samples provided below test without deploying to OpenFaaS
#
#CONFIG='config.json'

#
## FAILURE CASES :Invalid Inputs
#
#handle('')
#handle('"test":"ok"')
#handle('{"test":"ok"}')
#handle('{"data":"ok"}')

#
## FAILURE CASES :Unhandled Events
# 
# Standard : UserLogoutSessionEvent
#handle('{"id":"17e1027a-c865-4354-9c21-e8da3df4bff9","source":"https://vcsa.pdotk.local/sdk","specversion":"1.0","type":"com.vmware.event.router/event","subject":"UserLogoutSessionEvent","time":"2020-04-14T00:28:36.455112549Z","data":{"Key":7775,"ChainId":7775,"CreatedTime":"2020-04-14T00:28:35.221698Z","UserName":"machine-b8eb9a7f","Datacenter":null,"ComputeResource":null,"Host":null,"Vm":null,"Ds":null,"Net":null,"Dvs":null,"FullFormattedMessage":"User machine-b8ebe7eb9a7f@127.0.0.1 logged out (login time: Tuesday, 14 April, 2020 12:28:35 AM, number of API invocations: 34, user agent: pyvmomi Python/3.7.5 (Linux; 4.19.84-1.ph3; x86_64))","ChangeTag":"","IpAddress":"127.0.0.1","UserAgent":"pyvmomi Python/3.7.5 (Linux; 4.19.84-1.ph3; x86_64)","CallCount":34,"SessionId":"52edf160927","LoginTime":"2020-04-14T00:28:35.071817Z"},"datacontenttype":"application/json"}')
# Eventex : vim.event.ResourceExhaustionStatusChangedEvent
#handle('{"id":"0707d7e0-269f-42e7-ae1c-18458ecabf3d","source":"https://vcsa.pdotk.local/sdk","specversion":"1.0","type":"com.vmware.event.router/eventex","subject":"vim.event.ResourceExhaustionStatusChangedEvent","time":"2020-04-14T00:20:15.100325334Z","data":{"Key":7715,"ChainId":7715,"CreatedTime":"2020-04-14T00:20:13.76967Z","UserName":"machine-bb9a7f","Datacenter":null,"ComputeResource":null,"Host":null,"Vm":null,"Ds":null,"Net":null,"Dvs":null,"FullFormattedMessage":"vCenter Log File System Resource status changed from Yellow to Green on vcsa.pdotk.local  ","ChangeTag":"","EventTypeId":"vim.event.ResourceExhaustionStatusChangedEvent","Severity":"info","Message":"","Arguments":[{"Key":"resourceName","Value":"storage_util_filesystem_log"},{"Key":"oldStatus","Value":"yellow"},{"Key":"newStatus","Value":"green"},{"Key":"reason","Value":" "},{"Key":"nodeType","Value":"vcenter"},{"Key":"_sourcehost_","Value":"vcsa.pdotk.local"}],"ObjectId":"","ObjectType":"","ObjectName":"","Fault":null},"datacontenttype":"application/json"}')

#
## SUCCESS CASES
#
# Standard : VmPoweredOnEvent
#handle('{"id":"453120cd-3d19-4c43-aadc-df0cdbce3887","source":"https://vcsa.pdotk.local/sdk","specversion":"1.0","type":"com.vmware.event.router/event","subject":"VmPoweredOnEvent","time":"2020-04-13T23:46:10.402531287Z","data":{"Key":7441,"ChainId":7438,"CreatedTime":"2020-04-13T23:46:09.387283Z","UserName":"Administrator","Datacenter":{"Name":"PKLAB","Datacenter":{"Type":"Datacenter","Value":"datacenter-3"}},"ComputeResource":{"Name":"esxi01.pdotk.local","ComputeResource":{"Type":"ComputeResource","Value":"domain-s29"}},"Host":{"Name":"esxi01.pdotk.local","Host":{"Type":"HostSystem","Value":"host-31"}},"Vm":{"Name":"Test VM","Vm":{"Type":"VirtualMachine","Value":"vm-33"}},"Ds":null,"Net":null,"Dvs":null,"FullFormattedMessage":"Test VM on esxi01.pdotk.local in PKLAB has powered on","ChangeTag":"","Template":false},"datacontenttype":"application/json"}')
# Standard : VmPoweredOffEvent
#handle('{"id":"d77a3767-1727-49a3-ac33-ddbdef294150","source":"https://vcsa.pdotk.local/sdk","specversion":"1.0","type":"com.vmware.event.router/event","subject":"VmPoweredOffEvent","time":"2020-04-14T00:33:30.838669841Z","data":{"Key":7825,"ChainId":7821,"CreatedTime":"2020-04-14T00:33:30.252792Z","UserName":"Administrator","Datacenter":{"Name":"PKLAB","Datacenter":{"Type":"Datacenter","Value":"datacenter-3"}},"ComputeResource":{"Name":"esxi01.pdotk.local","ComputeResource":{"Type":"ComputeResource","Value":"domain-s29"}},"Host":{"Name":"esxi01.pdotk.local","Host":{"Type":"HostSystem","Value":"host-31"}},"Vm":{"Name":"Test VM","Vm":{"Type":"VirtualMachine","Value":"vm-33"}},"Ds":null,"Net":null,"Dvs":null,"FullFormattedMessage":"Test VM on  esxi01.pdotk.local in PKLAB is powered off","ChangeTag":"","Template":false},"datacontenttype":"application/json"}')
