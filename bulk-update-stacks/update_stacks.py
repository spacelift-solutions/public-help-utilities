import requests
import logging
import spacelift_auth


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

CONFIG = {
    "spacelift_url": "https://<your-spacelift-org>.app.spacelift.io/graphql",

    # update this object with whatever settings you'd like to update.

    # example attributes:

    # "default_branch": "main",
    # "namespace": "jubranNassar",
    # "worker_pool": "<private-worker-pool-id>", if using public worker pool, you can ignore this field
    # "provider": "GITHUB_ENTERPRISE",  # Or GITLAB
    # "terraform_version": "1.9.1"
}


def make_graphql_request(query, variables=None):
    try:
        response = requests.post(
            CONFIG["spacelift_url"],
            json={"query": query, "variables": variables or {}},
            headers={
                "Authorization": f"Bearer {spacelift_auth.get_spacelift_auth_token()}"},
            timeout=30
        )
        response.raise_for_status()
        return response.json()
    except requests.RequestException as e:
        logger.error(f"GraphQL request failed: {e}")
        raise


def update_stacks():
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
            """
    }

    # Get list of all stacks
    try:
        response = make_graphql_request(QUERIES["get_stacks"])

        if "errors" in response:
            logger.error(f"GraphQL errors: {response['errors']}")
            return

        stacks_list = [item["id"] for item in response["data"]["stacks"]]
        logger.info(f"Found {len(stacks_list)} stacks to update")

    except requests.RequestException as e:
        logger.error(f"Request failed: {e}")
        return
    except KeyError as e:
        logger.error(f"Unexpected response structure: {e}")
        return

    # Update each stack
    successful_updates = 0
    failed_updates = 0

    for stack in stacks_list:
        try:
            # Get stack details
            stack_data = make_graphql_request(
                QUERIES["get_stack"], variables={"id": stack})["data"]["stack"]

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
            update_response = make_graphql_request(
                QUERIES["update_stack"], variables=variables)

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
