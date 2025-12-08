# Spacelift Token Validator

A Go-based command-line tool for validating JWT tokens stored in Kubernetes secrets, specifically designed for Spacelift self-hosted installations. This utility helps verify that tokens are properly formatted, not expired, and contains valid claims.

## Features

- 🔐 **Kubernetes Integration**: Fetches tokens directly from Kubernetes secrets via kubectl
- ✅ **Format Validation**: Verifies proper JWT structure (header.payload.signature)
- 📝 **Claim Decoding**: Decodes and displays all standard and custom JWT claims
- ⏰ **Expiration Checking**: Validates token expiration and shows time until expiry
- 🎯 **Spacelift-Specific**: Designed for Spacelift self-hosted token validation
- 📊 **Clear Output**: Human-readable validation results with status indicators

## Prerequisites

- Go 1.21 or higher
- kubectl installed and configured
- Access to a Kubernetes cluster with the target secret

## Quick Start

Download, build, and run in one command:

```bash
curl -sL https://raw.githubusercontent.com/spacelift-solutions/public-help-utilities/main/token-validator/main.go -o main.go && go build -o token-validator main.go && ./token-validator --namespace {your-namespace}
```

## Installation

### Build from source

```bash
cd token-validator
go build -o token-validator main.go
```

### Run directly

```bash
go run main.go --secret <secret-name> [options]
```

## Usage

### Basic Usage

```bash
# Validate a token from a secret in the default namespace
./token-validator --secret spacelift-token

# Specify namespace and secret key
./token-validator --secret spacelift-token --namespace spacelift --key jwt-token
```

### Command-line Options

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--secret` | Kubernetes secret name | - | ✓ |
| `--namespace` | Kubernetes namespace | `default` | |
| `--key` | Key in the secret containing the JWT | `token` | |
| `--verbose` | Show verbose output including full token | `false` | |

### Examples

#### Validate token in spacelift namespace

```bash
./token-validator --secret spacelift-self-hosted-token --namespace spacelift
```

#### Validate with custom secret key

```bash
./token-validator --secret my-secret --key auth-token --namespace production
```

#### Show full token in output

```bash
./token-validator --secret spacelift-token --verbose
```

## Output

The tool provides comprehensive validation results:

```
Fetching secret 'spacelift-token' from namespace 'default'...
✓ Secret fetched successfully

======================================================================
SPACELIFT TOKEN VALIDATION RESULTS
======================================================================
✓ Token Status: VALID

✓ Format: Valid JWT structure

Token Header:
  Algorithm: RS256
  Type: JWT
  Key ID: spacelift-key-1

Token Claims:
  Issuer (iss): https://spacelift.example.com
  Subject (sub): spacelift-worker
  Audience (aud): spacelift-api
  JWT ID (jti): abc123-def456

Token Timing:
  Issued At: 2024-01-15T10:30:00Z
    └─ 5h30m ago
  Expires At: 2025-01-15T10:30:00Z
    └─ Valid for 8760h ✓

Additional Claims:
  worker_pool_id: pool-abc123
  account_name: my-spacelift-account
  permissions: [admin worker]

======================================================================
```

### Exit Codes

- `0`: Token is valid and not expired
- `1`: Token is invalid, expired, or failed to fetch

## Common Issues

### kubectl not found or not configured

Ensure kubectl is installed and you have a valid kubeconfig:

```bash
kubectl version --client
kubectl config current-context
```

### Secret not found

Verify the secret exists in the specified namespace:

```bash
kubectl get secrets -n <namespace>
kubectl describe secret <secret-name> -n <namespace>
```

### Key not found in secret

List available keys in the secret:

```bash
kubectl get secret <secret-name> -n <namespace> -o jsonpath='{.data}'
```

Then use the `--key` flag to specify the correct key name.

### Permission denied

Ensure your Kubernetes user/service account has permission to read secrets:

```bash
kubectl auth can-i get secrets --namespace <namespace>
```

## Validation Details

The tool performs the following validations:

1. **Format Check**: Verifies the token has three parts (header.payload.signature) separated by dots
2. **Base64 Decoding**: Validates that header and payload are properly base64url encoded
3. **JSON Structure**: Ensures header and payload contain valid JSON
4. **Expiration (exp)**: Checks if the token has expired based on the `exp` claim
5. **Not Before (nbf)**: Validates the token is not being used before its `nbf` time
6. **Claim Display**: Shows all standard JWT claims (iss, sub, aud, exp, nbf, iat, jti) and any custom claims

**Note**: This tool does NOT verify the token signature. Signature verification requires access to the signing key/certificate and is typically performed by the service consuming the token.

## Troubleshooting

### Token shows as expired

If your token is expired, you'll need to generate a new one. For Spacelift self-hosted installations, consult your Spacelift configuration for token generation procedures.

### Cannot decode token

If the token cannot be decoded, it may be:
- Not a valid JWT token
- Corrupted or incorrectly stored in the secret
- Encoded with an unexpected encoding (not base64url)

### Missing claims

If expected claims are missing from the output:
- Verify the token was generated with the correct claims
- Check if the token is intended for the use case you're validating
- Ensure the token hasn't been truncated or modified

## Security Considerations

- This tool reads sensitive secrets from your Kubernetes cluster
- The `--verbose` flag will display the complete token in output - use with caution
- Ensure proper RBAC is configured to limit access to secrets
- This tool should only be used by authorized personnel for troubleshooting purposes

## Contributing

This is an open-source tool maintained by the Spacelift Solutions team. Issues and pull requests are welcome.

## License

See LICENSE file in the repository root.
