import os
import logging
import json
import urllib.request
import sys


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

CONFIG = {
    "spacelift_url": os.environ.get("SPACELIFT_ENDPOINT") if os.environ.get("SPACELIFT_ENDPOINT", False) else logger.error("SPACELIFT_ENDPOINT environment variable is not set. Please set it to your Spacelift API endpoint."),
    "spacelift_api_key_id": f"{os.environ.get("SPACELIFT_API_KEY_ID")}" if os.environ.get("SPACELIFT_API_KEY_ID", False) else logger.error("SPACELIFT_API_KEY_ID environment variable is not set. Please set it to your Spacelift API Key ID."),
    "spacelift_api_key_secret": f"{os.environ.get("SPACELIFT_API_KEY_SECRET")}" if os.environ.get("SPACELIFT_API_KEY_SECRET", False) else logger.error("SPACELIFT_API_KEY_SECRET environment variable is not set. Please set it to your Spacelift API Key Secret."),
}

QUERIES = {
    "reslug_stack": """
            mutation($id:ID!) {
                stackReslug(id:$id) {	
                    id
                }
            }
            """,
    "get_jwt": """
        mutation GetSpaceliftToken($keyId: ID!, $keySecret: String!) {
            apiKeyUser(id: $keyId, secret: $keySecret) {
                id
                jwt
            }
        }
        """
}


class CustomError:
    def __init__(self, description, dump):
        self.description = description
        self.dump = dump

    def __call__(self, e=None):
        error_message = f"""
{self.description}


{json.dumps(self.dump, indent=2)}
        """
        print(error_message)
        if e is not None:
            print(e)
        raise Exception(f"{self.description}")


def query_api(query: str, check_data=None, variables: dict = None, err: CustomError = None) -> dict:
    try:
        if "GetSpaceliftToken" in query:
            headers = {
                "Content-Type": "application/json",
            }
        else:
            headers = {
                "Content-Type": "application/json",
                "Authorization": f"Bearer {get_spacelift_auth_token()}"
            }

        data = {
            "query": query,
        }

        if variables is not None:
            data["variables"] = variables

        if data is not None:
            req = urllib.request.Request(
                CONFIG["spacelift_url"], json.dumps(data).encode('utf-8'), headers)

        with urllib.request.urlopen(req) as response:
            resp = json.loads(response.read().decode('utf-8'))

        if "errors" in resp:
            err = CustomError(
                description="GraphQL request failed",
                dump=resp["errors"][0]["message"] if resp["errors"] else resp
            )

        if check_data is not None:
            def ensure_data(_resp, check):
                # Create an array of keys
                keys = [s for s in check.split(".?") if s]
                if len(keys) == 0:
                    return

                # Ensure the first key exists in the response
                this_key = keys[0]
                if this_key not in _resp:
                    err.code = f"{err.code}_{this_key.upper()}_MISSING"
                    err.dump = resp  # Keep this as the main resp for the error
                    err(query)
                # Delete the first key from the array and call ensure_data again
                del keys[0]
                ensure_data(_resp[this_key], '.?'.join(keys))
            ensure_data(resp, check_data)
        return resp
    except Exception as e:
        err(e)


def get_spacelift_auth_token():
    API_KEY_ID = CONFIG["spacelift_api_key_id"]
    API_KEY_SECRET = CONFIG["spacelift_api_key_secret"]

    variables = {
        "keyId": API_KEY_ID,
        "keySecret": API_KEY_SECRET
    }

    response = query_api(QUERIES["get_jwt"], variables=variables)

    if response['data']['apiKeyUser']['jwt'] is not None:
        API_TOKEN = response['data']['apiKeyUser']['jwt']
        return API_TOKEN


def reslug_stack(stack_id: str):
    """
    Reslug a specific stack by ID
    """
    try:
        logger.info(f"Reslugging stack: {stack_id}")

        variables = {
            "id": stack_id
        }

        # Call the reslug mutation
        response = query_api(
            QUERIES["reslug_stack"],
            variables=variables,
            check_data="data.?stackReslug",
            err=CustomError("Failed to reslug stack", None)
        )

        # Check for errors in the response
        if "errors" in response:
            logger.error(
                f"Failed to reslug stack {stack_id}: {response['errors'][0]['message']}")
            return False
        else:
            logger.info(f"Successfully reslugged stack: {stack_id}")
            return True

    except Exception as e:
        logger.error(f"Failed to reslug stack {stack_id}: {e}")
        return False


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python reslug_stack.py <stack_id>")
        print("Example: python reslug_stack.py my-stack-id")
        sys.exit(1)

    stack_id = sys.argv[1]
    success = reslug_stack(stack_id)

    if success:
        logger.info("Stack reslug completed successfully")
        sys.exit(0)
    else:
        logger.error("Stack reslug failed")
        sys.exit(1)
