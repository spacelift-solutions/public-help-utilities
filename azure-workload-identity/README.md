# Azure Workload Identity Setup for Spacelift Workers

This directory contains a robust, production-ready script for setting up Azure Workload Identity for Spacelift worker pools running on Azure Kubernetes Service (AKS).

## Overview

Azure Workload Identity allows Kubernetes workloads (like Spacelift workers) to authenticate with Azure services using managed identities, eliminating the need for storing service principal credentials as secrets.

## What This Script Does

The `setup-workload-identity.sh` script automates the complete setup process:

1. **Validates Prerequisites** - Ensures all required tools are installed and authenticated
2. **Enables Workload Identity** - Configures the AKS cluster with OIDC issuer and workload identity support  
3. **Creates Managed Identity** - Sets up a User Assigned Managed Identity in Azure
4. **Assigns Permissions** - Grants the specified Azure role to the managed identity
5. **Creates Federation** - Links the managed identity to the Kubernetes service account
6. **Configures Service Account** - Annotates and labels the Kubernetes service account

## Prerequisites

Before running the script, ensure you have:

### Required Tools
- **Azure CLI** (`az`) - For Azure resource management
- **kubectl** - For Kubernetes cluster access
- **jq** - For JSON parsing (used internally by the script)

The script will validate these tools are installed and provide installation instructions if any are missing.

### Authentication Requirements
- **Azure CLI Authentication** - Must be logged in (`az login`)
- **Kubernetes Authentication** - kubectl must be configured for your AKS cluster
- **Sufficient Permissions** - Your Azure account needs permissions to:
  - Modify AKS cluster settings
  - Create managed identities
  - Assign roles
  - Create federated credentials

## Usage

### Basic Usage
```bash
./setup-workload-identity.sh -g <resource-group> -c <cluster-name> -s <service-account> -l <location>
```

### Example
```bash
./setup-workload-identity.sh \
  -g "rg-spacelift-prod" \
  -c "aks-spacelift-cluster" \
  -s "spacelift-worker-sa" \
  -l "East US 2"
```

### Required Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `-g, --resource-group` | Azure resource group containing your AKS cluster | `rg-spacelift-prod` |
| `-c, --cluster-name` | Name of your AKS cluster | `aks-spacelift-cluster` |
| `-s, --service-account` | Kubernetes service account name for Spacelift workers | `spacelift-worker-sa` |
| `-l, --location` | Azure region where resources will be created | `East US 2` |

### Optional Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `-n, --namespace` | `spacelift-worker-pool-controller-system` | Kubernetes namespace for the service account |
| `-i, --identity-name` | `spacelift-worker-identity` | Name for the Azure managed identity |
| `-f, --federated-name` | `spacelift-federated-credential` | Name for the federated identity credential |
| `-r, --role` | `Contributor` | Azure role to assign to the managed identity |
| `--force` | false | Skip the confirmation prompt (useful for automation) |
| `-h, --help` | - | Show help message |
| `-v, --version` | - | Show script version |

## Script Features

### 🛡️ Comprehensive Validation
- Verifies all required tools are installed with helpful installation instructions
- Validates Azure CLI authentication and shows current subscription details
- Confirms kubectl connectivity and displays current cluster context
- Checks if target namespace and service account exist

### 📋 Detailed Configuration Display
Before making any changes, the script shows:
- Current Azure subscription, tenant, and user information
- Current kubectl context, cluster, and user details  
- Complete configuration summary of what will be created/modified
- Clear warnings about the scope of changes

### ✅ User-Friendly Interface
- Clear, colorful output with emojis for easy reading
- Comprehensive help documentation with examples
- Robust error messages with actionable solutions
- Interactive confirmation (can be bypassed with `--force`)

### 🔒 Security Best Practices
- Validates authentication before making any changes
- Shows exactly what permissions will be granted and where
- Requires explicit confirmation of destructive operations
- Uses least-privilege principles with configurable role assignment

## After Running the Script

Once the script completes successfully, you'll need to:

### 1. Update Your Spacelift WorkerPool

Patch your WorkerPool to use the configured service account:

```bash
kubectl patch workerpool <your-worker-pool-name> \
  -n spacelift-worker-pool-controller-system \
  --type='merge' \
  -p='{"spec":{"pod":{"serviceAccountName":"<service-account-name>","labels":{"azure.workload.identity/use":"true"}}}}'
```

### 2. Configure Terraform Azure Provider

In your Spacelift stacks, configure the Azure provider to use workload identity:

```hcl
provider "azurerm" {
  features {}
  use_aks_workload_identity = true
  subscription_id          = "<your-subscription-id>"
}
```

### 3. Verify Environment Variables

The workload identity webhook automatically injects these environment variables into your worker pods:
- `AZURE_CLIENT_ID` - The managed identity client ID
- `AZURE_TENANT_ID` - Your Azure tenant ID
- `AZURE_FEDERATED_TOKEN_FILE` - Path to the federated token file
- `AZURE_AUTHORITY_HOST` - Azure authority endpoint

## Troubleshooting

### Common Issues

**"Not logged in to Azure CLI"**
```bash
az login
# or for service principal authentication:
az login --service-principal -u <app-id> -p <password> --tenant <tenant>
```

**"No kubectl context is set"**
```bash
az aks get-credentials --resource-group <rg-name> --name <cluster-name>
```

**"Namespace does not exist"**
The script will warn if the target namespace doesn't exist but will continue. Ensure you have the correct namespace for your Spacelift worker pool controller.

**"Service account not found"**
The script will attempt to annotate/label the service account. If it doesn't exist, you may need to create it or check your Spacelift worker pool configuration.

### Validation Steps

After setup, verify the configuration:

```bash
# Check the managed identity
az identity show --name <identity-name> --resource-group <resource-group>

# Check the service account annotations
kubectl get serviceaccount <service-account> -n <namespace> -o yaml

# Check federated credentials
az identity federated-credential list --identity-name <identity-name> --resource-group <resource-group>
```

## Security Considerations

- **Scope of Permissions**: By default, the managed identity gets `Contributor` role on the entire resource group. Consider using more restrictive roles if possible.
- **Identity Lifecycle**: The managed identity persists after workload deletion. Clean up unused identities to maintain security hygiene.
- **Token Lifetime**: Federated tokens have limited lifetimes and are automatically rotated by the workload identity system.

## Script Versioning

The script includes version information accessible via:
```bash
./setup-workload-identity.sh --version
```

Current version: 1.0.0

## Contributing

When modifying this script:
1. Update the version number in the script header
2. Test thoroughly with various authentication scenarios  
3. Update this README with any new features or requirements
4. Ensure all error paths provide actionable guidance

## Resources

- [Azure Workload Identity Documentation](https://azure.github.io/azure-workload-identity/)
- [AKS Workload Identity Guide](https://docs.microsoft.com/en-us/azure/aks/workload-identity-overview)
- [Spacelift Worker Pool Documentation](https://docs.spacelift.io/concepts/worker-pools)