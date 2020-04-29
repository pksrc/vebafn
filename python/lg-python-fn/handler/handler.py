import sys, json, os
import urllib3
import requests
import traceback
from .util import Logger, FaaSResponse
#if you add other libraries, make sure to add them to the requirements.txt file

if(os.getenv("insecure_ssl")):
    # Surpress SSL warnings
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
l = Logger()

###
# Any Global variables or  Paths and Endpoints
# EDIT AS REQUIRED
###
API_URL='https://events.pagerduty.com/v2/enqueue'
CONFIG='/var/openfaas/secrets/config'
### --------

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
        ### --------

    def do_something(self,obj):
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
        l.log('TITLE', 'Doing something here:')
        try:
            resp = self.session.post(API_URL,json=obj)
            resp.raise_for_status()
            resp_body = json.loads(resp.text)
            return FaaSResponse('200', 'Successfully invoked PagerDuty API! dedup_key for this request: {0}'.format(resp_body['dedup_key']))
        except requests.HTTPError as err:
            return FaaSResponse('500', 'Could not invoke PagerDuty API > HTTPError: {0}'.format(err))
        ### --------

def handle(req):
    
    # Load the Events that function gets from vCenter through the Event Router
    l.log('TITLE', 'Reading Cloud Event:')
    l.log('INFO', 'Event > ',req)
    try:
        cevent = json.loads(req)
    except json.JSONDecodeError as err:
        res = FaaSResponse('400', 'Invalid JSON > JSONDecodeError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return

    # Load the Config File
    l.log('TITLE', 'Reading Configuration file:')
    l.log('INFO', 'Config File > ',CONFIG)
    try: 
        with open(CONFIG, 'r') as configfile:
            config = json.load(configfile)
    except json.JSONDecodeError as err:
        res = FaaSResponse('400', 'Invalid JSON > JSONDecodeError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return
    except OSError as err:
        res = FaaSResponse('500', 'Could not read configuration > OSError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return

    #Validate CloudEvent and Configuration for mandatory fields
    l.log('TITLE', 'Validating Input data')
    l.log('INFO', 'Event > ', json.dumps(cevent, indent=4, sort_keys=True))
    l.log('INFO', 'Config > ', json.dumps(config, indent=4, sort_keys=True))
    try:
        ###
        # Map the CloudEvent data and build the PagerDuty Event API Request body
        # EDIT AS REQUIRED
        ###
        #CloudEvent - simple validation
        event = cevent['data']
        
        #Config - checking for required fields
        routingkey=config['routing_key']
        event_action=config['event_action']
        
        obj = {
                'payload': {
                    'summary': event['FullFormattedMessage'],
                    'timestamp': event['CreatedTime'],
                    'source': cevent['source'],
                    'severity': 'info',
                    'component': event['Vm']['Name'],
                    'group': event['Host']['Name'],
                    'class': cevent['subject']
                },
                'client': 'VMware Event Broker Appliance',
                'client_url': cevent['source'],
                'routing_key': routingkey,
                'event_action': event_action
            }
        l.log('INFO', 'Successfully built API request body > ', json.dumps(obj, indent=4, sort_keys=True))
        ###--------
    except KeyError as err:
        res = FaaSResponse('400', 'Invalid JSON, required key not found > KeyError: {0}'.format(err))
        traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
        print(json.dumps(vars(res)))
        return
    except TypeError as err:
        res = FaaSResponse('400','Invalid JSON, missing required data > TypeError: {0}'.format(err))
        traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
        print(json.dumps(vars(res)))
        return

    # Make the Rest Api Call
    s=requests.Session()
    if(os.getenv("insecure_ssl")):
        s.verify=False
    
    ###
    # Map the CloudEvent data and build the PagerDuty Event API Request body
    # EDIT AS REQUIRED
    ###
    l.log('TITLE', 'Attempting HTTP POST:')
    try:
        ucn = YourClassName(s)
        res = ucn.do_something(obj)
        print(json.dumps(vars(res)))
    except Exception as err:
        res = FaaSResponse('500', 'Unexpected error occurred > Exception: {0}'.format(err))
        traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
        print(json.dumps(vars(res)))
    ###--------  
    s.close()

    return