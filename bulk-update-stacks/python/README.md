# Spacelift Stack Bulk Update Tool

A Python script for bulk updating Spacelift stacks

## Prerequisites

- Python 3.6 or higher

## Environment Variables

Set the following environment variables before running the script:

```bash
export SPACELIFT_DOMAIN="your-spacelift-domain"
export SPACELIFT_API_KEY_ID="your-api-key-id"
export SPACELIFT_API_KEY_SECRET="your-api-key-secret"
```

## Configuration

The script uses a `CONFIG` dictionary to define update settings.

Example configuration in the script:

```python
CONFIG = {
    "worker_pool": "your-worker-pool-id",
    # Add other settings as needed
}
```

## Usage

1. Set your environment variables
2. Configure the `CONFIG` dictionary with desired settings
3. Run the script:

```bash
python update_stacks.py
```

## What It Does

1. **Authentication**: Obtains a JWT token using your API credentials
2. **Stack Discovery**: Fetches all stacks from your Spacelift organization
3. **Stack Details**: Retrieves current configuration for each stack
4. **Bulk Update**: Updates each stack with new configuration while preserving existing settings
5. **Reporting**: Provides summary of successful and failed updates

## Current Update Operations

The script currently performs these updates on each stack:

- Assigns a new worker pool
- Updates Terraform workflow tool to OpenTofu
- Preserves all existing stack settings (branch, repository, namespace, etc.)

## Limitations

- **Terraform stacks only**: This script only works with terraform stacks

## Customization

To customize what gets updated:

1. Modify the `CONFIG` dictionary with your desired settings
2. Add or remove fields as needed for your use case
3. run the script

## Troubleshooting

- Ensure environment variables are set correctly
- Verify API key permissions
- Check Spacelift domain configuration
- Review logs for specific error messages
- Validate worker pool IDs exist in your organization

## Example Output

```
INFO:__main__:Found 25 stacks to update
INFO:__main__:Updating stack: stack-1...
INFO:__main__:Successfully updated stack: stack-1
INFO:__main__:Updating stack: stack-2...
INFO:__main__:Successfully updated stack: stack-2
...
INFO:__main__:Update complete: 23 successful, 2 failed
```
