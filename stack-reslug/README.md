# Spacelift Stack Reslug Tool

A simple Python script to reslug Spacelift stacks using the Spacelift GraphQL API.

## What is Stack Reslugging?

Stack reslugging in Spacelift regenerates the stack's slug (URL-friendly identifier) based on the current stack name. This can be useful when:

## Prerequisites

- Python 3.6 or higher

## Environment Variables

Set the following environment variables with your Spacelift credentials:

```bash
export SPACELIFT_ENDPOINT="https://your-spacelift-domain.app.spacelift.io/graphql"
export SPACELIFT_API_KEY_ID="your-api-key-id"
export SPACELIFT_API_KEY_SECRET="your-api-key-secret"
```

**Note:** Your Spacelift endpoint should be the full GraphQL API URL (e.g., `https://mycompany.app.spacelift.io/graphql`).

## Usage

### Basic Usage

```bash
python reslug_stack.py <stack_id>
```

### Examples

```bash
# Reslug a specific stack
python reslug_stack.py my-terraform-stack

# Reslug a stack with a UUID-style ID
python reslug_stack.py 01234567-89ab-cdef-0123-456789abcdef
```

## Command Line Options

| Argument | Description | Required |
|----------|-------------|----------|
| `stack_id` | The ID of the stack to reslug | Yes |

## Exit Codes

- `0`: Success - Stack was reslugged successfully
- `1`: Failure - Stack reslug failed or invalid usage

### Common Issues

**Authentication Error:**
```
SPACELIFT_ENDPOINT environment variable is not set
```
**Solution:** Ensure all required environment variables are set correctly.

**Stack Not Found:**
```
Failed to reslug stack: Stack not found
```
**Solution:** Verify the stack ID exists and you have access to it.

**Permission Denied:**
```
Failed to reslug stack: Insufficient permissions
```
**Solution:** Ensure your API key has the necessary permissions to modify stacks.
