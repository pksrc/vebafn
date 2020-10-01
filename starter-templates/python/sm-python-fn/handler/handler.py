# Inspired from contribution by Michael Gasch https://github.com/embano1/of-echo/
import json

def handle(req):
    """handle a request to the function
    Args:
        req (str): request body
    """
    
    print(f'Event (raw) > {req}')
    try:
        cevent = json.loads(req)
        print(f'Event (JSON) > {json.dumps(cevent, indent=4, sort_keys=True)}')
    except json.JSONDecodeError as err:
        print('Invalid JSON > JSONDecodeError: {0}'.format(err))

    return
