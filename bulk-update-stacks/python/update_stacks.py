import os
import logging
import json
import urllib.request


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

CONFIG = {
    "spacelift_url": f"https://{os.environ.get("SPACELIFT_DOMAIN")}.app.spacelift.io/graphql" if os.environ.get("SPACELIFT_DOMAIN", False) else logger.error("SPACELIFT_DOMAIN environment variable is not set. Please set it to your Spacelift domain."),
    "spacelift_api_key_id": f"{os.environ.get("SPACELIFT_API_KEY_ID")}" if os.environ.get("SPACELIFT_API_KEY_ID", False) else logger.error("SPACELIFT_API_KEY_ID environment variable is not set. Please set it to your Spacelift API Key ID."),
    "spacelift_api_key_secret": f"{os.environ.get("SPACELIFT_API_KEY_SECRET")}" if os.environ.get("SPACELIFT_API_KEY_SECRET", False) else logger.error("SPACELIFT_API_KEY_SECRET environment variable is not set. Please set it to your Spacelift API Key Secret."),

    # update this object with whatever settings you'd like to update.

    # example attributes:

    # "default_branch": "main",
    # "namespace": "your-github/gitlab namespace",  # e.g. "spacelift"
    # if using public worker pool, you can ignore this field
    # "worker_pool": "your-worker-pool-id",
    # "provider": "GITHUB_ENTERPRISE",  # Or GITLAB
    # "terraform_version": "1.9.1"
}

QUERIES = {
    "get_stacks": """
            query {
                stacks{
                id
                }
            }
            """,
    "get_stack": """
            query($id:ID!) {
                stack(id: $id) {
                    repository
                    branch
                    projectRoot
                    description
                    space
                    administrative
                    namespace
                    provider
                    labels
                    terraformVersion
                    workerPool {
                        id
                        space	
                    }
                    vcsIntegration {
                        id
                    }
                }
            }
            """,
    "update_stack": """
            mutation UpdateStack($stackId: ID!, $input: StackInput!) {
                stackUpdate(id: $stackId, input: $input) {
                    id
                    __typename
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


def update_stacks():
    try:
        response = query_api(QUERIES["get_stacks"], check_data="data.?stacks", err=CustomError(
            "Failed to fetch stacks", None))

        stacks_list = [item["id"] for item in response["data"]["stacks"]]
        logger.info(f"Found {len(stacks_list)} stacks to update")
    except KeyError as e:
        logger.error(f"Unexpected response structure: {e}")
        return

    # Update each stack
    successful_updates = 0
    failed_updates = 0

    for stack in stacks_list:
        try:
            # Get stack details
            stack_data = query_api(
                QUERIES["get_stack"], variables={"id": stack}, check_data="data.?stack", err=CustomError(
                    "Failed to fetch stack details", None))["data"]["stack"]
            # Build update variables
            variables = {
                "stackId": stack,
                "input": {
                    "administrative": stack_data["administrative"],
                    "branch": stack_data["branch"],
                    "name": stack,
                    "namespace": stack_data["namespace"],
                    "provider": stack_data["provider"],
                    "repository": stack_data["repository"],
                    "projectRoot": stack_data["projectRoot"],
                    "description": stack_data["description"],
                    "space": stack_data["space"],
                    "vcsIntegrationId": stack_data["vcsIntegration"]["id"] if stack_data["vcsIntegration"] else None,

                    # example of bulk updating all stacks to use a new worker pool:
                    "workerPool": CONFIG["worker_pool"],
                    "labels": stack_data["labels"],
                    "vendorConfig": {
                        "terraform": {
                            "version": stack_data["terraformVersion"],
                            "workflowTool": "OPEN_TOFU"
                        }
                    }
                }
            }

            # Update the stack
            logger.info(f"Updating stack: {stack}...")
            update_response = query_api(
                QUERIES["update_stack"], variables=variables, check_data="data.?stackUpdate", err=CustomError(
                    "Failed to update stack", None))
            # Check for errors in the update response
            if "errors" in update_response:
                logger.error(
                    f"Failed to update stack {stack}: {update_response['errors'][0]['message']}")
                failed_updates += 1
            else:
                logger.info(f"Successfully updated stack: {stack}")
                successful_updates += 1

        except Exception as e:
            logger.error(f"Failed to update stack {stack}: {e}")
            failed_updates += 1

    # Print summary
    logger.info(
        f"Update complete: {successful_updates} successful, {failed_updates} failed")

    if failed_updates > 0:
        logger.warning(
            f"{failed_updates} stacks failed to update. Check logs above for details.")


if __name__ == "__main__":
    update_stacks()
