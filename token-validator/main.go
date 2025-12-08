package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// JWTHeader represents the JWT header
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid,omitempty"`
}

// JWTPayload represents the JWT payload with standard and custom claims
type JWTPayload struct {
	Iss       string                 `json:"iss,omitempty"`
	Sub       string                 `json:"sub,omitempty"`
	Aud       interface{}            `json:"aud,omitempty"`
	Exp       int64                  `json:"exp,omitempty"`
	Nbf       int64                  `json:"nbf,omitempty"`
	Iat       int64                  `json:"iat,omitempty"`
	Jti       string                 `json:"jti,omitempty"`
	ExtraData map[string]interface{} `json:"-"`
}

// K8sSecret represents the structure of a Kubernetes secret
type K8sSecret struct {
	Data map[string]string `json:"data"`
}

// ValidationResult holds the results of JWT validation
type ValidationResult struct {
	Valid           bool
	FormatValid     bool
	Expired         bool
	Header          *JWTHeader
	Payload         *JWTPayload
	RawToken        string
	ExpiresAt       time.Time
	IssuedAt        time.Time
	NotBefore       time.Time
	TimeUntilExpiry time.Duration
	Errors          []string
}

func main() {
	namespace := flag.String("namespace", "default", "Kubernetes namespace")
	secretName := flag.String("secret", "spacelift-shared", "Kubernetes secret name")
	secretKey := flag.String("key", "token", "Key in the secret containing the JWT token")
	verbose := flag.Bool("verbose", false, "Show verbose output including full token")
	flag.Parse()

	if *secretName == "" {
		fmt.Println("Error: --secret flag is required")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Fetching secret '%s' from namespace '%s'...\n", *secretName, *namespace)

	// Fetch secret from Kubernetes
	token, err := getTokenFromK8sSecret(*namespace, *secretName, *secretKey)
	if err != nil {
		fmt.Printf("✗ Failed to fetch secret: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Secret fetched successfully")
	fmt.Println()

	// Validate the JWT token
	result := validateJWT(token)

	// Print results
	printValidationResult(result, *verbose)

	// Exit with appropriate code
	if !result.Valid || result.Expired {
		os.Exit(1)
	}
}

// getTokenFromK8sSecret fetches a JWT token from a Kubernetes secret
func getTokenFromK8sSecret(namespace, secretName, key string) (string, error) {
	cmd := exec.Command("kubectl", "get", "secret", secretName, "-n", namespace, "-o", "json")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("kubectl error: %w\nStderr: %s", err, stderr.String())
	}

	var secret K8sSecret
	if err := json.Unmarshal(stdout.Bytes(), &secret); err != nil {
		return "", fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	encodedToken, exists := secret.Data[key]
	if !exists {
		availableKeys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			availableKeys = append(availableKeys, k)
		}
		return "", fmt.Errorf("key '%s' not found in secret. Available keys: %v", key, availableKeys)
	}

	// Kubernetes stores secret data as base64, decode it
	tokenBytes, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 token: %w", err)
	}

	return string(tokenBytes), nil
}

// validateJWT performs comprehensive JWT validation
func validateJWT(token string) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		FormatValid: false,
		RawToken:    token,
		Errors:      make([]string, 0),
	}

	// Check JWT format (3 parts separated by dots)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		result.Valid = false
		result.FormatValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JWT format: expected 3 parts, got %d", len(parts)))
		return result
	}
	result.FormatValid = true

	// Decode header
	header, err := decodeJWTHeader(parts[0])
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to decode header: %v", err))
	} else {
		result.Header = header
	}

	// Decode payload
	payload, err := decodeJWTPayload(parts[1])
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to decode payload: %v", err))
	} else {
		result.Payload = payload

		// Check expiration
		if payload.Exp > 0 {
			result.ExpiresAt = time.Unix(payload.Exp, 0)
			result.TimeUntilExpiry = time.Until(result.ExpiresAt)

			if time.Now().Unix() > payload.Exp {
				result.Expired = true
				result.Valid = false
				result.Errors = append(result.Errors, "Token has expired")
			}
		}

		// Parse issued at time
		if payload.Iat > 0 {
			result.IssuedAt = time.Unix(payload.Iat, 0)
		}

		// Parse not before time
		if payload.Nbf > 0 {
			result.NotBefore = time.Unix(payload.Nbf, 0)
			if time.Now().Unix() < payload.Nbf {
				result.Valid = false
				result.Errors = append(result.Errors, "Token not yet valid (nbf claim)")
			}
		}
	}

	return result
}

// decodeJWTHeader decodes the JWT header
func decodeJWTHeader(encodedHeader string) (*JWTHeader, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encodedHeader)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	var header JWTHeader
	if err := json.Unmarshal(decoded, &header); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	return &header, nil
}

// decodeJWTPayload decodes the JWT payload
func decodeJWTPayload(encodedPayload string) (*JWTPayload, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	// First unmarshal into a map to capture all claims
	var allClaims map[string]interface{}
	if err := json.Unmarshal(decoded, &allClaims); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Then unmarshal into our struct for standard claims
	var payload JWTPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Store extra claims that aren't in our standard fields
	payload.ExtraData = make(map[string]interface{})
	standardClaims := map[string]bool{
		"iss": true, "sub": true, "aud": true, "exp": true,
		"nbf": true, "iat": true, "jti": true,
	}

	for key, value := range allClaims {
		if !standardClaims[key] {
			payload.ExtraData[key] = value
		}
	}

	return &payload, nil
}

// printValidationResult prints the validation results in a formatted way
func printValidationResult(result ValidationResult, verbose bool) {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("SPACELIFT TOKEN VALIDATION RESULTS")
	fmt.Println(strings.Repeat("=", 70))

	// Overall status
	if result.Valid && !result.Expired {
		fmt.Println("✓ Token Status: VALID")
	} else if result.Expired {
		fmt.Println("✗ Token Status: EXPIRED")
	} else {
		fmt.Println("✗ Token Status: INVALID")
	}
	fmt.Println()

	// Format validation
	if result.FormatValid {
		fmt.Println("✓ Format: Valid JWT structure")
	} else {
		fmt.Println("✗ Format: Invalid JWT structure")
	}
	fmt.Println()

	// Header information
	if result.Header != nil {
		fmt.Println("Token Header:")
		fmt.Printf("  Algorithm: %s\n", result.Header.Alg)
		fmt.Printf("  Type: %s\n", result.Header.Typ)
		if result.Header.Kid != "" {
			fmt.Printf("  Key ID: %s\n", result.Header.Kid)
		}
		fmt.Println()
	}

	// Payload information
	if result.Payload != nil {
		fmt.Println("Token Claims:")

		if result.Payload.Iss != "" {
			fmt.Printf("  Issuer (iss): %s\n", result.Payload.Iss)
		}
		if result.Payload.Sub != "" {
			fmt.Printf("  Subject (sub): %s\n", result.Payload.Sub)
		}
		if result.Payload.Aud != nil {
			fmt.Printf("  Audience (aud): %v\n", result.Payload.Aud)
		}
		if result.Payload.Jti != "" {
			fmt.Printf("  JWT ID (jti): %s\n", result.Payload.Jti)
		}

		fmt.Println()
		fmt.Println("Token Timing:")

		if !result.IssuedAt.IsZero() {
			fmt.Printf("  Issued At: %s\n", result.IssuedAt.Format(time.RFC3339))
			fmt.Printf("    └─ %s ago\n", time.Since(result.IssuedAt).Round(time.Second))
		}

		if !result.NotBefore.IsZero() {
			fmt.Printf("  Not Before: %s\n", result.NotBefore.Format(time.RFC3339))
		}

		if !result.ExpiresAt.IsZero() {
			fmt.Printf("  Expires At: %s\n", result.ExpiresAt.Format(time.RFC3339))
			if result.Expired {
				fmt.Printf("    └─ Expired %s ago ✗\n", time.Since(result.ExpiresAt).Round(time.Second))
			} else {
				fmt.Printf("    └─ Valid for %s ✓\n", result.TimeUntilExpiry.Round(time.Second))
			}
		}

		// Print extra claims (Spacelift-specific or custom)
		if len(result.Payload.ExtraData) > 0 {
			fmt.Println()
			fmt.Println("Additional Claims:")
			for key, value := range result.Payload.ExtraData {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
		fmt.Println()
	}

	// Print errors if any
	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range result.Errors {
			fmt.Printf("  ✗ %s\n", err)
		}
		fmt.Println()
	}

	// Print raw token if verbose
	if verbose {
		fmt.Println("Raw Token:")
		fmt.Printf("  %s\n", result.RawToken)
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 70))
}
