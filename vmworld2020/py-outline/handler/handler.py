
def handle(req):
    """handle a request to the function
    Args:
        req (str): Cloud Event as String
    """

    print(f'Event (raw) > {req}')
    #parse event
    

    PAGERDUTY_CONFIG = '/var/openfaas/secrets/pdconfig'
    #read the config
    

    #implement business logic
    PAGERDUTY_API = 'https://events.pagerduty.com/v2/enqueue'
    #make a POST Rest API request to pagerduty
    


    return
