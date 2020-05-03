import sys, json, os
import urllib3
from urllib.parse import parse_qsl, unquote
import requests
import traceback
import hashlib, hmac
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
CONFIG='/var/openfaas/secrets/vcconfig'
### --------

def handle(req):
    
    ###
    # Example of the payload received from Slack
    #
    # token = 'pXxmdXfTp8HfNa9Z4v74mslg'
    # team_id = 'T9D21FD7Z'
    # team_domain = 'pklack' 
    # channel_id = 'C017763BZGX' 
    # channel_name = 'vmworld2020'
    # user_id = 'U9BDPPEQ1'
    # user_name = 'partheeban.kandasamy'
    # command = 'echo'
    # response_url = 'https%3A%2F%2Fhooks.slack.com%2Fcommands%2FT9D21FD7Z%2F1266291674740%2FeRXh0iOFOSe75kTbE1KRGiY7'
    # trigger_id = '1258627504981.319069523271.6f1d5e972b786e0979f734485f07cc27'
    # text = "createvm"
    ##
    
    ###
    #  Load the Events that function gets from vCenter through the Event Router
    ##
    l.log('TITLE', 'Reading Slack slash command request:')
    l.log('INFO', 'Request > ',req)
    try:
        obj = dict(parse_qsl(req))
        print(obj)
    except Exception as err:
        res = FaaSResponse('400', 'Invalid JSON > JSONDecodeError: {0}'.format(err))
        print(json.dumps(vars(res)))
        return
    
    s=requests.Session()
    if(os.getenv("insecure_ssl")):
        s.verify=False

    # Verifying Signature of the Slack payload
    #data = unquote(req)
    # data = urllib.parse.urlencode(urllib.parse.unquote(raw_string))
    #l.log('INFO', 'data > ',data)
    signature = str(os.getenv('Http_X_Slack_Signature'))
    timestamp = str(os.getenv('Http_X_Slack_Request_Timestamp'))
    format_req = str.encode(f"v0:{timestamp}:{req}")
    SLACK_SECRET = '097d3f67da972dd18e133db109902879'
    encoded_secret = str.encode(SLACK_SECRET)
    request_hash = hmac.new(encoded_secret, format_req, hashlib.sha256).hexdigest()
    calculated_signature = f"v0={request_hash}"
    l.log('INFO', 'calculated_signature > ',calculated_signature)
    l.log('INFO', 'provided_signature > ',signature)
    if not hmac.compare_digest(calculated_signature, signature):
        try:
            errauthresp = s.post(obj['response_url'],json=json.loads('{"text": "Invalid Auth","response_type": "ephemeral"}'))
            errauthresp.raise_for_status()
            return
        except requests.HTTPError as err:
            return json.dumps(vars(FaaSResponse('500', 'HTTPError: {0}'.format(err))))

    ###
    # Acknowledging user's command invocation by responding back to Slack request
    # the response url is in the payload received 
    ###
    l.log('TITLE', 'Attempting HTTP POST:{0}'.format(obj['response_url']))
    try:
        ackObj = {
            'text': f'ACK - Processing Command ```{obj["text"]}```', 
            'response_type': 'ephemeral'
        }
        ackresp = s.post(obj['response_url'],json=ackObj)
        ackresp.raise_for_status()
    except requests.HTTPError as err:
        return json.dumps(vars(FaaSResponse('500', 'HTTPError: {0}'.format(err))))
    
    ###
    # Invoking the appropriate VM function based on the command received
    # For security purposes, we'll send a shared key which will be verified in the VM function (this could be passed as a config)
    ###
    l.log('INFO', 'Attempting Command:{0}'.format(obj['text']))
    #https://www.guidgenerator.com/online-guid-generator.aspx 
    obj['key']='2F232EB71D584140B9529460340FCFE4' 
    if('echo' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-echo', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Echo Function Failed > HTTPError: {0}'.format(err))))

    elif('spawn' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-createvm', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Create VM Function Failed > HTTPError: {0}'.format(err))))
       
    elif('clonetemplate' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-vmclonetemplate', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Clone Template Function Failed > HTTPError: {0}'.format(err))))
 
    elif('clone' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-clonevm', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Clone VM Function Failed > HTTPError: {0}'.format(err))))

    elif('poweron' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-poweronvm', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'PowerOn VM Function Failed > HTTPError: {0}'.format(err))))

    elif('poweroff' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-poweroffvm', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'PowerOff VM Function Failed > HTTPError: {0}'.format(err))))

    elif('reboot' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-rebootvm', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Reboot VM Function Failed > HTTPError: {0}'.format(err))))
    
    elif('nuke' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-deletevm', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Delete VM Function Failed > HTTPError: {0}'.format(err))))

    elif('transform' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-setvm', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Transform VM Function Failed > HTTPError: {0}'.format(err))))

    elif('invoke' in obj['text']):
        try:
            cmdresp = s.post('http://gateway.openfaas:8080/async-function/powercli-danger', json=obj)
            cmdresp.raise_for_status()
            print(cmdresp.text)
        except requests.HTTPError as err:
            traceback.print_exc(limit=1, file=sys.stderr) #providing traceback since it helps debug the exact key that failed
            return json.dumps(vars(FaaSResponse('500', 'Invoke Function Failed > HTTPError: {0}'.format(err))))

    else:
        try:
            errresp = s.post(obj['response_url'],json=json.loads('{"text": "ERR - No matching command","response_type": "in_channel"}'))
            errresp.raise_for_status()
        except requests.HTTPError as err:
            return json.dumps(vars(FaaSResponse('500', 'HTTPError: {0}'.format(err))))

    ###--------  
    s.close()
    return json.dumps(vars(FaaSResponse('200', 'Successfully Executed Command')))
