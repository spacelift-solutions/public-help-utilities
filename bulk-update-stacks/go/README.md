# Spacelift Bulk Update Tool

A Go-based command-line tool for performing bulk updates on stacks in Spacelift. This tool allows you to efficiently update multiple stacks concurrently with configurable parameters.

## Features

- 🚀 **Concurrent Processing**: Updates multiple stacks simultaneously using goroutines
- 🎯 **Terraform Stack Filtering**: Automatically filters and processes only Terraform stacks
- 🔧 **Environment Variable Configuration**: Flexible configuration through environment variables
- 📊 **Detailed Reporting**: Comprehensive summary with success/failure statistics
- 🔐 **Secure Authentication**: Uses Spacelift API keys for secure access
- ⚡ **GraphQL API Integration**: Efficient communication with Spacelift's GraphQL API

## Prerequisites

- Go 1.19 or higher
- Valid Spacelift account with API access
- Spacelift API key with appropriate permissions

## Configuration

The tool uses environment variables for configuration. Set the following variables before running:

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `SPACELIFT_DOMAIN` | Your Spacelift domain (without .app.spacelift.io) |
| `SPACELIFT_API_KEY_ID` | Your Spacelift API key ID |
| `SPACELIFT_API_KEY_SECRET` | Your Spacelift API key secret |

### Optional Update Parameters

Configure any of the following to update stack properties:

| Variable | Description | Example |
|----------|-------------|---------|
| `SPACELIFT_ADMINISTRATIVE` | Set stack as administrative | `true` or `false` |
| `SPACELIFT_NAME` | Update stack name | `my-updated-stack` |
| `SPACELIFT_BRANCH` | Update branch | `main` |
| `SPACELIFT_NAMESPACE` | Update namespace | `my-namespace` |
| `SPACELIFT_PROVIDER` | Update provider | `terraform` |
| `SPACELIFT_REPOSITORY` | Update repository | `my-org/my-repo` |
| `SPACELIFT_REPOSITORY_URL` | Update repository URL(for RAW_GIT provider) | `https://github.com/my-org/my-repo` |
| `SPACELIFT_PROJECT_ROOT` | Update project root path | `/infrastructure` |
| `SPACELIFT_SPACE` | Update space | `production` |
| `SPACELIFT_LABELS` | Add labels (comma-separated) | `env:prod,team:platform` |
| `SPACELIFT_WORKER_POOL_ID` | Update worker pool | `worker-pool-123` |

## Usage

### Basic Usage

```bash
# Set required environment variables
export SPACELIFT_DOMAIN="your-domain"
export SPACELIFT_API_KEY_ID="your-key-id"
export SPACELIFT_API_KEY_SECRET="your-key-secret"


### Example: Update All Stacks with New Labels

```bash
export SPACELIFT_DOMAIN="mycompany"
export SPACELIFT_API_KEY_ID="ak-12345"
export SPACELIFT_API_KEY_SECRET="secret-67890"
export SPACELIFT_LABELS="updated:2024,team:platform"

go run update_stacks.go
```

### Example: Move All Stacks to Production Space

```bash
export SPACELIFT_DOMAIN="mycompany"
export SPACELIFT_API_KEY_ID="ak-12345"
export SPACELIFT_API_KEY_SECRET="secret-67890"
export SPACELIFT_SPACE="production-12345687"

go run update_stacks.go
```

### Example: Update Worker Pool for All Stacks

```bash
export SPACELIFT_DOMAIN="mycompany"
export SPACELIFT_API_KEY_ID="ak-12345"
export SPACELIFT_API_KEY_SECRET="secret-67890"
export SPACELIFT_WORKER_POOL_ID="new-worker-pool-id"

go run update_stacks.go
```

## Output

The tool provides real-time feedback and a comprehensive summary:

```
Authenticating with Spacelift...
✓ Authentication successful
Retrieving all stacks...
✓ Retrieved 45 stacks
Starting bulk update...
Processing stack: stack-1
Processing stack: stack-2
✓ Successfully updated stack: stack-1
✓ Successfully updated stack: stack-2
...

============================================================
BULK UPDATE SUMMARY
============================================================
Total stacks processed: 42
Successful updates: 40
Failed updates: 2
Execution time: 15.234s
Success rate: 95.24%

✗ Failed to update stacks:
  - stack-error-1: GraphQL error: Stack not found
  - stack-error-2: GraphQL error: Insufficient permissions
============================================================
```

## Important Notes

- ⚠️ **Currently, only Terraform stacks are supported**

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   - Verify your API key ID and secret are correct
   - Ensure your API key has sufficient permissions
   - Check that your domain is correct (without the .app.spacelift.io suffix)

2. **No Stacks Found**
   - Verify your API key has access to view stacks
   - Check if you have any Terraform stacks in your account

3. **Permission Errors**
   - Ensure your API key has stack update permissions
   - Check space-specific permissions if updating space configurations

### Debug Mode

For additional debugging, you can modify the code to include more verbose logging or add the `-v` flag when running with `go run -v main.go`.
