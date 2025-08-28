#!/bin/bash

# Azure Workload Identity Setup for Spacelift Workers
set -e

# Script configuration
SCRIPT_NAME="$(basename "$0")"
VERSION="1.0.0"

# Default values
DEFAULT_SERVICE_ACCOUNT_NAMESPACE="spacelift-worker-pool-controller-system"
DEFAULT_IDENTITY_NAME="spacelift-worker-identity"
DEFAULT_FEDERATED_CREDENTIAL_NAME="spacelift-federated-credential"
DEFAULT_ROLE="Contributor"

# Variables to be set by parameters
RESOURCE_GROUP=""
CLUSTER_NAME=""
SERVICE_ACCOUNT_NAME=""
LOCATION=""
SERVICE_ACCOUNT_NAMESPACE="$DEFAULT_SERVICE_ACCOUNT_NAMESPACE"
IDENTITY_NAME="$DEFAULT_IDENTITY_NAME"
FEDERATED_CREDENTIAL_NAME="$DEFAULT_FEDERATED_CREDENTIAL_NAME"
ROLE="$DEFAULT_ROLE"
FORCE=false

# Function to display usage information
show_usage() {
    cat << EOF
$SCRIPT_NAME - Azure Workload Identity Setup for Spacelift Workers

USAGE:
    $SCRIPT_NAME -g <resource-group> -c <cluster-name> -s <service-account> -l <location> [options]

REQUIRED PARAMETERS:
    -g, --resource-group    Azure resource group name
    -c, --cluster-name      AKS cluster name  
    -s, --service-account   Kubernetes service account name
    -l, --location          Azure region (e.g., "West US 2")

OPTIONAL PARAMETERS:
    -n, --namespace         Service account namespace (default: $DEFAULT_SERVICE_ACCOUNT_NAMESPACE)
    -i, --identity-name     Managed identity name (default: $DEFAULT_IDENTITY_NAME)
    -f, --federated-name    Federated credential name (default: $DEFAULT_FEDERATED_CREDENTIAL_NAME)
    -r, --role             Azure role to assign (default: $DEFAULT_ROLE)
    --force                Skip confirmation prompt
    -h, --help             Show this help message
    -v, --version          Show version information

EXAMPLE:
    $SCRIPT_NAME -g "rg-aks-prod" -c "aks-prod-cluster" -s "spacelift-worker" -l "West US 2"

EOF
}

# Function to show version
show_version() {
    echo "$SCRIPT_NAME version $VERSION"
}

# Function to parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -g|--resource-group)
                RESOURCE_GROUP="$2"
                shift 2
                ;;
            -c|--cluster-name)
                CLUSTER_NAME="$2"
                shift 2
                ;;
            -s|--service-account)
                SERVICE_ACCOUNT_NAME="$2"
                shift 2
                ;;
            -l|--location)
                LOCATION="$2"
                shift 2
                ;;
            -n|--namespace)
                SERVICE_ACCOUNT_NAMESPACE="$2"
                shift 2
                ;;
            -i|--identity-name)
                IDENTITY_NAME="$2"
                shift 2
                ;;
            -f|--federated-name)
                FEDERATED_CREDENTIAL_NAME="$2"
                shift 2
                ;;
            -r|--role)
                ROLE="$2"
                shift 2
                ;;
            --force)
                FORCE=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--version)
                show_version
                exit 0
                ;;
            *)
                echo "❌ Error: Unknown parameter '$1'"
                echo "Use '$SCRIPT_NAME --help' for usage information."
                exit 1
                ;;
        esac
    done
}

# Function to validate required parameters
validate_parameters() {
    local errors=0
    
    if [[ -z "$RESOURCE_GROUP" ]]; then
        echo "❌ Error: Resource group (-g|--resource-group) is required"
        errors=$((errors + 1))
    fi
    
    if [[ -z "$CLUSTER_NAME" ]]; then
        echo "❌ Error: Cluster name (-c|--cluster-name) is required"
        errors=$((errors + 1))
    fi
    
    if [[ -z "$SERVICE_ACCOUNT_NAME" ]]; then
        echo "❌ Error: Service account name (-s|--service-account) is required"
        errors=$((errors + 1))
    fi
    
    if [[ -z "$LOCATION" ]]; then
        echo "❌ Error: Location (-l|--location) is required"
        errors=$((errors + 1))
    fi
    
    if [[ $errors -gt 0 ]]; then
        echo ""
        echo "Use '$SCRIPT_NAME --help' for usage information."
        exit 1
    fi
}

# Function to check prerequisites and gather context information
check_prerequisites() {
    echo "🔍 Checking prerequisites and authentication status..."
    echo ""
    
    # Check if az CLI is installed
    if ! command -v az &> /dev/null; then
        echo "❌ Error: Azure CLI (az) is not installed or not in PATH"
        echo "Please install Azure CLI: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
        echo ""
        echo "Installation options:"
        echo "  # macOS (Homebrew):"
        echo "  brew install azure-cli"
        echo ""
        echo "  # Linux (apt):"
        echo "  curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash"
        echo ""
        echo "  # Windows (winget):"
        echo "  winget install Microsoft.AzureCLI"
        exit 1
    fi
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        echo "❌ Error: kubectl is not installed or not in PATH"
        echo "Please install kubectl: https://kubernetes.io/docs/tasks/tools/install-kubectl/"
        echo ""
        echo "Installation options:"
        echo "  # macOS (Homebrew):"
        echo "  brew install kubectl"
        echo ""
        echo "  # Linux:"
        echo "  curl -LO \"https://dl.k8s.io/release/\$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl\""
        echo "  sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl"
        echo ""
        echo "  # Windows (winget):"
        echo "  winget install Kubernetes.kubectl"
        exit 1
    fi
    
    # Check if jq is installed (needed for JSON parsing)
    if ! command -v jq &> /dev/null; then
        echo "❌ Error: jq is not installed or not in PATH"
        echo "jq is required for parsing JSON output from Azure CLI"
        echo ""
        echo "Installation options:"
        echo "  # macOS (Homebrew):"
        echo "  brew install jq"
        echo ""
        echo "  # Linux (apt):"
        echo "  sudo apt-get install jq"
        echo ""
        echo "  # Linux (yum):"
        echo "  sudo yum install jq"
        echo ""
        echo "  # Windows (winget):"
        echo "  winget install jqlang.jq"
        exit 1
    fi
    
    # Check Azure authentication and get account details
    if ! AZURE_ACCOUNT=$(az account show --output json 2>/dev/null); then
        echo "❌ Error: Not logged in to Azure CLI"
        echo "Please run 'az login' to authenticate with Azure"
        echo ""
        echo "To login:"
        echo "  az login"
        echo "  # or for service principal:"
        echo "  az login --service-principal -u <app-id> -p <password> --tenant <tenant>"
        exit 1
    fi
    
    # Extract Azure account information
    AZURE_SUBSCRIPTION_ID=$(echo "$AZURE_ACCOUNT" | jq -r '.id')
    AZURE_SUBSCRIPTION_NAME=$(echo "$AZURE_ACCOUNT" | jq -r '.name')
    AZURE_TENANT_ID=$(echo "$AZURE_ACCOUNT" | jq -r '.tenantId')
    AZURE_USER_NAME=$(echo "$AZURE_ACCOUNT" | jq -r '.user.name // "Service Principal"')
    AZURE_USER_TYPE=$(echo "$AZURE_ACCOUNT" | jq -r '.user.type // "servicePrincipal"')
    
    # Check kubectl authentication and context
    if ! KUBECTL_CONTEXT=$(kubectl config current-context 2>/dev/null); then
        echo "❌ Error: No kubectl context is set"
        echo "Please configure kubectl to connect to your AKS cluster"
        echo ""
        echo "To configure kubectl for AKS:"
        echo "  az aks get-credentials --resource-group <rg-name> --name <cluster-name>"
        exit 1
    fi
    
    # Test kubectl authentication by trying to access cluster info
    if ! kubectl cluster-info &> /dev/null; then
        echo "❌ Error: kubectl is not authenticated or cluster is not accessible"
        echo "Current context: $KUBECTL_CONTEXT"
        echo ""
        echo "Please ensure:"
        echo "  1. You have valid credentials for the cluster"
        echo "  2. The cluster is running and accessible"
        echo "  3. Your kubectl context is correctly configured"
        echo ""
        echo "To refresh AKS credentials:"
        echo "  az aks get-credentials --resource-group <rg-name> --name <cluster-name> --overwrite-existing"
        exit 1
    fi
    
    # Get additional kubectl context information
    KUBECTL_CLUSTER=$(kubectl config view --minify -o jsonpath='{.clusters[0].name}' 2>/dev/null || echo "Unknown")
    KUBECTL_USER=$(kubectl config view --minify -o jsonpath='{.users[0].name}' 2>/dev/null || echo "Unknown")
    KUBECTL_NAMESPACE=$(kubectl config view --minify -o jsonpath='{.contexts[0].context.namespace}' 2>/dev/null || echo "default")
    
    # Test if we can access the target namespace
    if ! kubectl get namespace "$SERVICE_ACCOUNT_NAMESPACE" &> /dev/null; then
        echo "⚠️  Warning: Namespace '$SERVICE_ACCOUNT_NAMESPACE' does not exist or is not accessible"
        echo "The script will attempt to access this namespace later. Ensure you have appropriate permissions."
        echo ""
    fi
    
    # Test if the service account exists (if namespace is accessible)
    if kubectl get namespace "$SERVICE_ACCOUNT_NAMESPACE" &> /dev/null; then
        if kubectl get serviceaccount "$SERVICE_ACCOUNT_NAME" -n "$SERVICE_ACCOUNT_NAMESPACE" &> /dev/null; then
            SA_EXISTS="Yes (will be updated)"
        else
            SA_EXISTS="No (will be created if it doesn't exist)"
        fi
    else
        SA_EXISTS="Unknown (namespace not accessible)"
    fi
    
    echo "✅ All prerequisites validated successfully"
    echo ""
}

# Function to display configuration and ask for confirmation
show_configuration() {
    echo "🔧 Azure Workload Identity Setup Configuration"
    echo "============================================="
    echo ""
    echo "🔐 Current Authentication Context:"
    echo "  Azure Subscription:   $AZURE_SUBSCRIPTION_NAME ($AZURE_SUBSCRIPTION_ID)"
    echo "  Azure Tenant:         $AZURE_TENANT_ID"
    echo "  Azure User:           $AZURE_USER_NAME ($AZURE_USER_TYPE)"
    echo "  Kubectl Context:      $KUBECTL_CONTEXT"
    echo "  Kubectl Cluster:      $KUBECTL_CLUSTER"
    echo "  Kubectl User:         $KUBECTL_USER"
    echo "  Kubectl Namespace:    $KUBECTL_NAMESPACE"
    echo ""
    echo "📋 Configuration Summary:"
    echo "  Resource Group:        $RESOURCE_GROUP"
    echo "  AKS Cluster:          $CLUSTER_NAME"
    echo "  Location:             $LOCATION"
    echo "  Service Account:      $SERVICE_ACCOUNT_NAME"
    echo "  Namespace:            $SERVICE_ACCOUNT_NAMESPACE"
    echo "  Service Account Exists: $SA_EXISTS"
    echo "  Managed Identity:     $IDENTITY_NAME"
    echo "  Federated Credential: $FEDERATED_CREDENTIAL_NAME"
    echo "  Azure Role:           $ROLE"
    echo ""
    echo "🚀 This script will:"
    echo "  1. Enable Workload Identity and OIDC issuer on the AKS cluster"
    echo "  2. Create a User Assigned Managed Identity named '$IDENTITY_NAME'"
    echo "  3. Grant '$ROLE' role to the identity on resource group '$RESOURCE_GROUP'"
    echo "  4. Create a federated identity credential for the service account"
    echo "  5. Annotate and label the Kubernetes service account '$SERVICE_ACCOUNT_NAME'"
    echo ""
    echo "⚠️  IMPORTANT WARNINGS:"
    echo "  • This will modify Azure resources in subscription: $AZURE_SUBSCRIPTION_NAME"
    echo "  • This will modify Kubernetes resources in cluster: $KUBECTL_CLUSTER"
    echo "  • Ensure you have sufficient permissions in both Azure and Kubernetes"
    echo "  • The managed identity will have '$ROLE' permissions on the entire resource group"
    echo ""
    
    if [[ "$FORCE" != "true" ]]; then
        echo "🤔 Do you want to proceed with these changes?"
        echo ""
        read -p "Type 'yes' to continue or any other key to cancel: " -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            echo "❌ Operation cancelled by user"
            exit 0
        fi
        echo ""
    fi
    
    echo "🎯 Starting Azure Workload Identity setup..."
    echo ""
}

# Parse command line arguments
parse_arguments "$@"

# Validate parameters
validate_parameters

# Check prerequisites
check_prerequisites

# Show configuration and get confirmation
show_configuration

# Use the subscription ID we already extracted
SUBSCRIPTION_ID="$AZURE_SUBSCRIPTION_ID"
echo "Using subscription: $AZURE_SUBSCRIPTION_NAME ($AZURE_SUBSCRIPTION_ID)"

# Step 1: Enable Workload Identity on the cluster
echo "Enabling Workload Identity on AKS cluster..."
az aks update \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --enable-workload-identity \
  --enable-oidc-issuer

# Step 1.5: Workload Identity webhook (managed by AKS automatically)
echo "✅ Azure Workload Identity webhook is managed by AKS"

# Step 2: Create a User Assigned Managed Identity
echo "Creating User Assigned Managed Identity..."
az identity create \
  --name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --location "$LOCATION"

# Step 3: Get identity details
IDENTITY_CLIENT_ID=$(az identity show \
  --name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --query clientId -o tsv)

IDENTITY_PRINCIPAL_ID=$(az identity show \
  --name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --query principalId -o tsv)

OIDC_ISSUER=$(az aks show \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --query oidcIssuerProfile.issuerUrl -o tsv)

echo "Identity Client ID: $IDENTITY_CLIENT_ID"
echo "Identity Principal ID: $IDENTITY_PRINCIPAL_ID"
echo "OIDC Issuer: $OIDC_ISSUER"

# Step 4: Grant the identity permissions to create resources
echo "Granting $ROLE role to the managed identity..."
az role assignment create \
  --assignee $IDENTITY_PRINCIPAL_ID \
  --role "$ROLE" \
  --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP"

# Step 5: Create federated identity credential
echo "Creating federated identity credential..."
az identity federated-credential create \
  --name "$FEDERATED_CREDENTIAL_NAME" \
  --identity-name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --issuer $OIDC_ISSUER \
  --subject "system:serviceaccount:$SERVICE_ACCOUNT_NAMESPACE:$SERVICE_ACCOUNT_NAME"

# Step 6: Annotate the service account
echo "Annotating the Kubernetes service account..."
kubectl annotate serviceaccount $SERVICE_ACCOUNT_NAME \
  --namespace $SERVICE_ACCOUNT_NAMESPACE \
  azure.workload.identity/client-id=$IDENTITY_CLIENT_ID \
  --overwrite

# Step 7: Label the service account
echo "Labeling the Kubernetes service account..."
kubectl label serviceaccount $SERVICE_ACCOUNT_NAME \
  --namespace $SERVICE_ACCOUNT_NAMESPACE \
  azure.workload.identity/use=true \
  --overwrite

echo ""
echo "✅ Workload Identity setup complete!"
echo ""
echo "Summary:"
echo "- Identity Name: $IDENTITY_NAME"
echo "- Client ID: $IDENTITY_CLIENT_ID"
echo "- Service Account: $SERVICE_ACCOUNT_NAMESPACE/$SERVICE_ACCOUNT_NAME"
echo "- Permissions: $ROLE on resource group $RESOURCE_GROUP"
echo ""
echo "Next Steps for Spacelift Integration:"
echo "1. Update your WorkerPool to use the annotated service account:"
echo "   kubectl patch workerpool my-amazing-worker-pool -n spacelift-worker-pool-controller-system --type='merge' -p='{\"spec\":{\"pod\":{\"serviceAccountName\":\"$SERVICE_ACCOUNT_NAME\",\"labels\":{\"azure.workload.identity/use\":\"true\"}}}}'"
echo ""
echo "2. Configure your Terraform Azure provider with:"
echo "   provider \"azurerm\" {"
echo "     features {}"
echo "     use_aks_workload_identity = true"
echo "     subscription_id          = \"$SUBSCRIPTION_ID\""
echo "   }"
echo ""
echo "3. The workload identity webhook will automatically inject these environment variables:"
echo "   - AZURE_CLIENT_ID"
echo "   - AZURE_TENANT_ID" 
echo "   - AZURE_FEDERATED_TOKEN_FILE"
echo "   - AZURE_AUTHORITY_HOST"
echo ""
echo "Your Spacelift workers can now authenticate to Azure using Workload Identity!"
echo "Test by running a Spacelift stack that creates Azure resources."