# Remote State Config Check

Spacelift supports External State access, see our docs [here](https://docs.spacelift.io/vendors/terraform/external-state-access.html) to read more on it.
Sometimes, configuring this can be tricky. This script will tell you what is wrong with your configuration (hopefully).

[Heres](https://support.spacelift.io/articles/9582913392-resolving-cannot-create-a-workspace-error-when-accessing-external-state) a useful help article on the topic as well.

### Pre-requisites

- Python 3.x or later
- [SpaceCTL](https://docs.spacelift.io/concepts/spacectl)

### Usage

1. Clone this repository
2. Login with SpaceCTL
  - `spacectl profile login`
3. Export your spacelift api token
  - `export SPACELIFT_API_TOKEN=$(spacectl profile export-token)`
  - Note: ensure the correct profile is selected with `spacectl profile select {profile_name}` if you have multiple profiles.
4. Run the script with the following options:
  - `SPACELIFT_HOSTNAME="{your_hostname}" SPACELIFT_ORGANIZATION="{your_org}" SPACELIFT_WORKSPACE="{your_workspace}" python state_check.py`

If you were using the following setup, this is how you would configure the above command:
```hcl
terraform {
  backend "remote" {
    hostname     = "spacelift.io" # this is the value of SPACELIFT_HOSTNAME
    organization = "apollorion" # this is the value of SPACELIFT_ORGANIZATION

    workspaces {
      name = "vpc" # this is the value of SPACELIFT_WORKSPACE
    }
  }
}
```

### Did it work?

If the script runs successfully, you will see the following message: 
```text
If you made it this far, your configuration is correct and should work with OpenTofu or Terraform.
```

If it didn't, it will tell you what it thinks is wrong with your configuration and you can try to fix it again or send the debug message to the SE team and we can help you.

#### Error Codes

Send this info to the SE youre currently working with for more information.

- `NO_WELL_KNOWN_FOUND`: We couldnt find your well-known configuration.
  - Ensure you have the correct values for `SPACELIFT_HOSTNAME`.
  - Ensure your token is correct is has the right permissions.
- `NO_WELL_KNOWN_FOUND_STATE.V3_MISSING`: We found your well-known config, but the response was bad.
- `ENTITLEMENTS_NOT_FOUND`: We couldnt hit your entitlements url.
  - Check if your organization is correct. It should the org id, which can be found in the URL.
- `ENTITLEMENTS_NOT_FOUND_DATA_MISSING`, `ENTITLEMENTS_NOT_FOUND_ATTRIBUTES_MISSING`, `ENTITLEMENTS_NOT_FOUND_STATE-STORAGE_MISSING`: We couldnt find the correct data in the entitlements response.
- `WORKSPACE_NOT_FOUND`: We couldnt find the given workspace.
    - Ensure you have the correct values for `SPACELIFT_WORKSPACE`.
    - The workspace should be the workspace ID, not the name.
- `WORKSPACE_NOT_FOUND_DATA_MISSING`, `WORKSPACE_NOT_FOUND_ID_MISSING`: We couldnt find the correct data in the workspace response.
- `STATE_NOT_FOUND` We found the organization, and the workspace, but the state was not found.
  - Try triggering the stack to generate some state.
- `STATE_NOT_FOUND_DATA_MISSING`, `STATE_NOT_FOUND_ATTRIBUTES_MISSING`, `STATE_NOT_FOUND_HOSTED-STATE-DOWNLOAD-URL_MISSING`: We couldnt find the correct data in the state response.
  - Try triggering the stack to generate some state.