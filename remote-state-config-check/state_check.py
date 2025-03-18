import urllib.request
import urllib.error
import json
import os

# Make sure required vars are set
required_vars = {
    "SPACELIFT_API_TOKEN": "This should be your api token you get from spacelift, you can get one from spacectl via `spacectl profile export-token`.",
    "SPACELIFT_HOSTNAME": "This should be what you set as the \"hostname\" in your terraform remote backend.",
    "SPACELIFT_ORGANIZATION": "This should be what you set as the \"organization\" in your terraform remote backend.",
    "SPACELIFT_WORKSPACE": "This should be what you set as the \"workspaces.name\" in your terraform remote backend."
}
missing_vars = []
for var in required_vars.keys():
    try:
        os.environ[var]
    except KeyError:
        missing_vars.append(var)

if len(missing_vars) > 0:
    print(f"Missing required environment variables: {missing_vars}")
    print("")
    for var in missing_vars:
        print(f"{var}: {required_vars[var]}")
    exit(1)

token = os.environ["SPACELIFT_API_TOKEN"]
hostname = os.environ["SPACELIFT_HOSTNAME"]
organization = os.environ["SPACELIFT_ORGANIZATION"]
workspace = os.environ["SPACELIFT_WORKSPACE"]

class CustomError:
    def __init__(self, description, code, dump):
        self.description = description
        self.code = code
        self.dump = dump

    def __call__(self, uri, e=None):
        error_message = f"""
{self.description}

Code: {self.code}

We attempted the following URL: {uri}
The last successful call told us:
{json.dumps(self.dump, indent=2)}
        """
        print(error_message)
        if e is not None:
            print(e)
        exit(1)

def call_api(uri, check_data=None, data=None, err: CustomError = None):
    try:
        headers = {
            'Content-Type': 'application/vnd.api+json',
            'Authorization': f"Bearer {token}"
        }

        if data is None:
            req = urllib.request.Request(uri, headers=headers)
        else:
            req = urllib.request.Request(uri, json.dumps(data).encode('utf-8'), headers)

        with urllib.request.urlopen(req) as response:
            resp = json.loads(response.read().decode('utf-8'))

        if check_data is not None:
            def ensure_data(resp, check):
                # Create an array of keys
                keys = [s for s in check.split(".?") if s]
                if len(keys) == 0:
                    return

                # Ensure the first key exists in the response
                this_key = keys[0]
                if this_key not in resp:
                    err.code = f"{err.code}_{this_key.upper()}_MISSING"
                    err.dump = resp
                    err(uri)
                # Delete the first key from the array and call ensure_data again
                del keys[0]
                ensure_data(resp[this_key], '.?'.join(keys))
            ensure_data(resp, check_data)

        return resp
    except Exception as e:
        err(uri, e)


def main():
    # Find the terraform well-known info
    well_known = call_api(
        uri=f"https://{hostname}/.well-known/terraform.json",
        check_data="state.v2",
        err=CustomError(
            description="Well known address not found, maybe your remote state hostname is incorrect?",
            code="NO_WELL_KNOWN_FOUND",
            dump={}
        )
    )
    state_v2_url = well_known["state.v2"]

    # Find the organization entitlements
    entitlements = call_api(
        uri=f"{state_v2_url}/organizations/{organization}/entitlement-set",
        check_data="data.?attributes.?state-storage",
        err=CustomError(
            description="No entitlements found, maybe your organization is incorrect?",
            code="ENTITLEMENTS_NOT_FOUND",
            dump=well_known
        )
    )

    # Find the workspace
    list_workspace = call_api(
        uri=f"{state_v2_url}/organizations/{organization}/workspaces/{workspace}",
        check_data="data.?id",
        err=CustomError(
            description="Your organization is correctly configured for external state, but the workspace is not found. Maybe your workspace is incorrect?",
            code="WORKSPACE_NOT_FOUND",
            dump=entitlements
        )
    )
    workspace_id = list_workspace["data"]["id"]

    # Get the current state version
    call_api(
        uri=f"{state_v2_url}/workspaces/{workspace_id}/current-state-version",
        check_data="data.?attributes.?hosted-state-download-url",
        err=CustomError(
            description="Your workspace is correctly configured for external state, but we found no state. Maybe you need to trigger the stack?",
            code="STATE_NOT_FOUND",
            dump=list_workspace
        )
    )

    print("If you made it this far, your configuration is correct and should work with OpenTofu or Terraform.")

if __name__ == "__main__":
    main()