package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/auth"
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Manage authentication clients",
	Long:  `Manage authentication clients for JWT and Signature authentication`,
}

var clientCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new authentication client",
	Long:  `Create a new authentication client with specified type and permissions`,
	RunE:  runClientCreate,
}

var clientListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all authentication clients",
	Long:  `List all authentication clients with their configuration`,
	RunE:  runClientList,
}

var clientShowCmd = &cobra.Command{
	Use:           "show [client_id]",
	Short:         "Show specific client details",
	Long:          `Show detailed configuration for a specific client`,
	Args:          cobra.ExactArgs(1),
	RunE:          runClientShow,
	SilenceErrors: true,
}

var clientRevokeCmd = &cobra.Command{
	Use:           "revoke [client_id]",
	Short:         "Revoke client access (set active=false)",
	Long:          `Revoke client access by setting active status to false`,
	Args:          cobra.ExactArgs(1),
	RunE:          runClientRevoke,
	SilenceErrors: true,
}

var clientActivateCmd = &cobra.Command{
	Use:           "activate [client_id]",
	Short:         "Activate client access (set active=true)",
	Long:          `Activate client access by setting active status to true`,
	Args:          cobra.ExactArgs(1),
	RunE:          runClientActivate,
	SilenceErrors: true,
}

var clientRegenerateCmd = &cobra.Command{
	Use:           "regenerate [client_id]",
	Short:         "Regenerate client secret key (HMAC only)",
	Long:          `Regenerate secret key for HMAC clients`,
	Args:          cobra.ExactArgs(1),
	RunE:          runClientRegenerate,
	SilenceErrors: true,
}

var clientDeleteCmd = &cobra.Command{
	Use:           "delete [client_id]",
	Short:         "Delete client permanently",
	Long:          `Permanently delete client from configuration`,
	Args:          cobra.ExactArgs(1),
	RunE:          runClientDelete,
	SilenceErrors: true,
}

var clientReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload all clients from config file",
	Long:  `Reload all authentication clients from config file to memory cache`,
	RunE:  runClientReload,
}

var clientGenerateSignCmd = &cobra.Command{
	Use:           "generatesign [client_id]",
	Short:         "Generate signature for testing",
	Long:          `Generate signature headers for testing API endpoints (HMAC clients only)`,
	Args:          cobra.ExactArgs(1),
	RunE:          runClientGenerateSign,
	SilenceErrors: true,
}

// Command flags
var (
	clientName        string
	clientType        string
	clientPermissions string
	clientKeyPath     string
	forceDelete       bool
	signMethod        string
	signPath          string
	withNonce         bool
)

func init() {
	// Add subcommands
	clientCmd.AddCommand(clientCreateCmd)
	clientCmd.AddCommand(clientListCmd)
	clientCmd.AddCommand(clientShowCmd)
	clientCmd.AddCommand(clientRevokeCmd)
	clientCmd.AddCommand(clientActivateCmd)
	clientCmd.AddCommand(clientRegenerateCmd)
	clientCmd.AddCommand(clientDeleteCmd)
	clientCmd.AddCommand(clientReloadCmd)
	clientCmd.AddCommand(clientGenerateSignCmd)

	// Create command flags
	clientCreateCmd.Flags().StringVarP(&clientName, "name", "n", "", "Client name (required)")
	clientCreateCmd.Flags().StringVarP(&clientType, "type", "t", "hmac", "Auth type: rsa or hmac (default: hmac)")
	clientCreateCmd.Flags().StringVarP(&clientPermissions, "permissions", "p", "read:health,read:ping", "Comma-separated permissions")
	clientCreateCmd.Flags().StringVarP(&clientKeyPath, "key-path", "k", "", "Public key path (required for RSA type)")
	clientCreateCmd.MarkFlagRequired("name")

	// Delete command flags
	clientDeleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force delete without confirmation")

	// Generate signature command flags
	clientGenerateSignCmd.Flags().StringVarP(&signMethod, "method", "m", "GET", "HTTP method (default: GET)")
	clientGenerateSignCmd.Flags().StringVarP(&signPath, "path", "p", "/v1/health", "API path (default: /v1/health)")
	clientGenerateSignCmd.Flags().BoolVarP(&withNonce, "with-nonce", "n", false, "Include nonce for replay attack prevention")

	// Add to root command
	rootCmd.AddCommand(clientCmd)
}

// generateClientID generates a random client ID
func generateClientID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateSecretKey generates a secure random secret key based on algorithm
func generateSecretKey() string {
	// Get key size based on algorithm
	keySize := getKeySize()
	bytes := make([]byte, keySize)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getKeySize returns optimal key size based on config algorithm
func getKeySize() int {
	algorithm := config.Get().Auth.Algorithm
	switch algorithm {
	case "HS256", "RS256":
		return 32 // 256 bits
	case "HS512", "RS512":
		return 64 // 512 bits
	default:
		return 32 // Default to 256 bits
	}
}

// saveConfig saves the updated configuration to file
func saveConfig(cfg *config.Config) error {
	file, err := os.OpenFile(".config.json", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %v", err)
	}

	return nil
}

// runClientCreate creates a new authentication client
func runClientCreate(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Validate auth type
	if clientType != "rsa" && clientType != "hmac" {
		return fmt.Errorf("invalid auth type: %s (must be 'rsa' or 'hmac')", clientType)
	}

	// Validate RSA key path
	if clientType == "rsa" && clientKeyPath == "" {
		return fmt.Errorf("key-path is required for RSA auth type")
	}

	// Check if key file exists for RSA
	if clientType == "rsa" {
		if _, err := os.Stat(clientKeyPath); os.IsNotExist(err) {
			return fmt.Errorf("public key file not found: %s", clientKeyPath)
		}
	}

	// Parse permissions
	permissions := strings.Split(clientPermissions, ",")
	for i, perm := range permissions {
		permissions[i] = strings.TrimSpace(perm)
	}

	// Generate client ID
	clientID := generateClientID()

	// Create client config
	newClient := config.ClientConfig{
		ClientID:    clientID,
		ClientName:  clientName,
		AuthType:    clientType,
		Permissions: permissions,
		Active:      true,
	}

	// Set auth-specific fields
	if clientType == "rsa" {
		newClient.KeyPath = clientKeyPath
	} else {
		newClient.SecretKey = generateSecretKey()
	}

	// DUAL UPDATE: 1. Add to memory cache first
	if err := auth.AddClient(newClient); err != nil {
		return fmt.Errorf("failed to add client to memory cache: %v", err)
	}

	// DUAL UPDATE: 2. Add to config file
	cfg.Auth.Clients = append(cfg.Auth.Clients, newClient)
	if err := saveConfig(cfg); err != nil {
		// Rollback: remove from memory cache if config save fails
		auth.RemoveClient(clientID)
		return fmt.Errorf("failed to save config (rolled back memory cache): %v", err)
	}

	// Display result
	fmt.Printf("✅ Client created successfully!\n\n")
	fmt.Printf("Client ID:    %s\n", clientID)
	fmt.Printf("Client Name:  %s\n", clientName)
	fmt.Printf("Auth Type:    %s\n", clientType)
	fmt.Printf("Permissions:  %s\n", strings.Join(permissions, ", "))
	fmt.Printf("Status:       active\n")

	if clientType == "hmac" {
		fmt.Printf("\n🔑 Secret Key: %s\n", newClient.SecretKey)
		fmt.Printf("\n⚠️  Save this secret key securely - it won't be shown again!\n")
	} else {
		fmt.Printf("Key Path:     %s\n", clientKeyPath)
	}

	fmt.Printf("\n✨ Client is immediately active - no server restart required!\n")

	return nil
}

// runClientList lists all authentication clients
func runClientList(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	if len(cfg.Auth.Clients) == 0 {
		fmt.Println("No authentication clients configured.")
		return nil
	}

	fmt.Printf("Authentication Clients (%d total):\n\n", len(cfg.Auth.Clients))
	fmt.Printf("%-32s %-20s %-8s %-8s %-30s\n", "CLIENT ID", "NAME", "TYPE", "STATUS", "PERMISSIONS")
	fmt.Printf("%-32s %-20s %-8s %-8s %-30s\n",
		strings.Repeat("-", 32),
		strings.Repeat("-", 20),
		strings.Repeat("-", 8),
		strings.Repeat("-", 8),
		strings.Repeat("-", 30))

	for _, client := range cfg.Auth.Clients {
		status := "active"
		if !client.Active {
			status = "revoked"
		}

		perms := strings.Join(client.Permissions, ",")
		if len(perms) > 30 {
			perms = perms[:27] + "..."
		}

		fmt.Printf("%-32s %-20s %-8s %-8s %-30s\n",
			client.ClientID,
			client.ClientName,
			client.AuthType,
			status,
			perms)
	}

	return nil
}

// runClientShow shows detailed client information
func runClientShow(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	clientID := args[0]

	// Find client
	var client *config.ClientConfig
	for _, c := range cfg.Auth.Clients {
		if c.ClientID == clientID {
			client = &c
			break
		}
	}

	if client == nil {
		fmt.Printf("❌ Client not found: %s\n", clientID)
		fmt.Printf("\nUse 'client list' to see all available clients.\n")
		return fmt.Errorf("client not found")
	}

	// Display client details
	fmt.Printf("Client Details:\n\n")
	fmt.Printf("Client ID:    %s\n", client.ClientID)
	fmt.Printf("Client Name:  %s\n", client.ClientName)
	fmt.Printf("Auth Type:    %s\n", client.AuthType)
	fmt.Printf("Status:       %s\n", map[bool]string{true: "active", false: "revoked"}[client.Active])
	fmt.Printf("Permissions:  %s\n", strings.Join(client.Permissions, ", "))

	if client.AuthType == "rsa" {
		fmt.Printf("Key Path:     %s\n", client.KeyPath)
	} else {
		fmt.Printf("Secret Key:   %s****** (hidden)\n", client.SecretKey[:8])
	}

	return nil
}

// runClientRevoke revokes client access
func runClientRevoke(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	clientID := args[0]

	// Find and update client
	found := false
	var updatedClient config.ClientConfig
	for i, client := range cfg.Auth.Clients {
		if client.ClientID == clientID {
			if !client.Active {
				fmt.Printf("Client %s (%s) is already revoked.\n", clientID, client.ClientName)
				return nil
			}
			cfg.Auth.Clients[i].Active = false
			updatedClient = cfg.Auth.Clients[i]
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ Client not found: %s\n", clientID)
		fmt.Printf("\nUse 'client list' to see all available clients.\n")
		return fmt.Errorf("client not found")
	}

	// DUAL UPDATE: 1. Update memory cache
	if err := auth.UpdateClient(updatedClient); err != nil {
		return fmt.Errorf("failed to update client in memory cache: %v", err)
	}

	// DUAL UPDATE: 2. Save config file
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Printf("✅ Client %s access revoked successfully (immediate effect).\n", clientID)

	return nil
}

// runClientActivate activates client access
func runClientActivate(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	clientID := args[0]

	// Find and update client
	found := false
	var updatedClient config.ClientConfig
	for i, client := range cfg.Auth.Clients {
		if client.ClientID == clientID {
			if client.Active {
				fmt.Printf("Client %s (%s) is already active.\n", clientID, client.ClientName)
				return nil
			}
			cfg.Auth.Clients[i].Active = true
			updatedClient = cfg.Auth.Clients[i]
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ Client not found: %s\n", clientID)
		fmt.Printf("\nUse 'client list' to see all available clients.\n")
		return fmt.Errorf("client not found")
	}

	// DUAL UPDATE: 1. Update memory cache
	if err := auth.UpdateClient(updatedClient); err != nil {
		return fmt.Errorf("failed to update client in memory cache: %v", err)
	}

	// DUAL UPDATE: 2. Save config file
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Printf("✅ Client %s access activated successfully (immediate effect).\n", clientID)

	return nil
}

// runClientRegenerate regenerates client secret key
func runClientRegenerate(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	clientID := args[0]

	// Find client
	found := false
	var updatedClient config.ClientConfig
	for i, client := range cfg.Auth.Clients {
		if client.ClientID == clientID {
			if client.AuthType != "hmac" {
				return fmt.Errorf("secret key regeneration is only available for HMAC clients")
			}

			// Generate new secret key
			newSecretKey := generateSecretKey()
			cfg.Auth.Clients[i].SecretKey = newSecretKey
			updatedClient = cfg.Auth.Clients[i]
			found = true

			fmt.Printf("✅ Secret key regenerated for client %s (%s)\n\n", clientID, client.ClientName)
			fmt.Printf("🔑 New Secret Key: %s\n", newSecretKey)
			fmt.Printf("\n⚠️  Save this secret key securely - it won't be shown again!\n")
			break
		}
	}

	if !found {
		fmt.Printf("❌ Client not found: %s\n", clientID)
		fmt.Printf("\nUse 'client list' to see all available clients.\n")
		return fmt.Errorf("client not found")
	}

	// DUAL UPDATE: 1. Update memory cache
	if err := auth.UpdateClient(updatedClient); err != nil {
		return fmt.Errorf("failed to update client in memory cache: %v", err)
	}

	// DUAL UPDATE: 2. Save config file
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Printf("\n✨ New secret key is immediately active - no server restart required!\n")

	return nil
}

// runClientDelete permanently deletes a client
func runClientDelete(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	clientID := args[0]

	// Find client
	var clientIndex = -1
	var clientName string
	for i, client := range cfg.Auth.Clients {
		if client.ClientID == clientID {
			clientIndex = i
			clientName = client.ClientName
			break
		}
	}

	if clientIndex == -1 {
		fmt.Printf("❌ Client not found: %s\n", clientID)
		fmt.Printf("\nUse 'client list' to see all available clients.\n")
		return fmt.Errorf("client not found")
	}

	// Confirmation unless force flag is used
	if !forceDelete {
		fmt.Printf("⚠️  Are you sure you want to permanently delete client %s (%s)? [y/N]: ", clientID, clientName)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Deletion cancelled.")
			return nil
		}
	}

	// DUAL UPDATE: 1. Remove from memory cache
	if err := auth.RemoveClient(clientID); err != nil {
		return fmt.Errorf("failed to remove client from memory cache: %v", err)
	}

	// DUAL UPDATE: 2. Remove from config file
	cfg.Auth.Clients = append(cfg.Auth.Clients[:clientIndex], cfg.Auth.Clients[clientIndex+1:]...)
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Printf("✅ Client %s (%s) deleted permanently (immediate effect).\n", clientID, clientName)

	return nil
}

// runClientReload reloads all clients from config file
func runClientReload(cmd *cobra.Command, args []string) error {
	if err := auth.ReloadAuth(); err != nil {
		return fmt.Errorf("failed to reload authentication system: %v", err)
	}

	fmt.Printf("✅ Authentication system reloaded from config file successfully.\n")

	return nil
}

// runClientGenerateSign generates signature for testing API endpoints
func runClientGenerateSign(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	clientID := args[0]

	// Find client
	var client *config.ClientConfig
	for _, c := range cfg.Auth.Clients {
		if c.ClientID == clientID {
			client = &c
			break
		}
	}

	if client == nil {
		fmt.Printf("❌ Client not found: %s\n", clientID)
		fmt.Printf("\nUse 'client list' to see all available clients.\n")
		return fmt.Errorf("client not found")
	}

	// Check if client is HMAC type
	if client.AuthType != "hmac" {
		fmt.Printf("❌ Signature generation is only supported for HMAC clients.\n")
		fmt.Printf("Client %s (%s) is using %s authentication.\n", clientID, client.ClientName, client.AuthType)
		return fmt.Errorf("unsupported auth type for signature generation")
	}

	// Check if client is active
	if !client.Active {
		fmt.Printf("⚠️  Warning: Client %s (%s) is currently inactive.\n", clientID, client.ClientName)
	}

	// Generate current timestamp
	timestamp := time.Now().Unix()

	// Generate nonce only if requested
	var nonce string
	if withNonce {
		nonceBytes := make([]byte, 16)
		rand.Read(nonceBytes)
		nonce = hex.EncodeToString(nonceBytes)
	}

	// Create signature payload
	payload := auth.SignaturePayload{
		ClientID:  clientID,
		Timestamp: timestamp,
		Nonce:     nonce,
		Method:    signMethod,
		Path:      signPath,
		Body:      "", // No body for GET requests
	}

	// Generate signature using the client's secret key
	signature, err := auth.GenerateSignature(payload, client.SecretKey)
	if err != nil {
		return fmt.Errorf("failed to generate signature: %v", err)
	}

	// Display results
	fmt.Printf("🔐 Signature Generated Successfully!\n\n")
	fmt.Printf("Client Details:\n")
	fmt.Printf("  Client ID:   %s\n", client.ClientID)
	fmt.Printf("  Client Name: %s\n", client.ClientName)
	fmt.Printf("  Status:      %s\n", map[bool]string{true: "active", false: "inactive"}[client.Active])
	fmt.Printf("\nRequest Details:\n")
	fmt.Printf("  Method:      %s\n", signMethod)
	fmt.Printf("  Path:        %s\n", signPath)
	fmt.Printf("  Timestamp:   %d\n", timestamp)
	if withNonce {
		fmt.Printf("  Nonce:       %s\n", nonce)
	}
	fmt.Printf("\nGenerated Headers:\n")
	fmt.Printf("  X-Client-ID: %s\n", clientID)
	fmt.Printf("  X-Timestamp: %d\n", timestamp)
	if withNonce {
		fmt.Printf("  X-Nonce:     %s\n", nonce)
	}
	fmt.Printf("  X-Signature: %s\n", signature)

	// Generate ready-to-use curl command
	fmt.Printf("\n📋 Ready-to-use curl command:\n")
	fmt.Printf("curl -s -X %s \\\n", signMethod)
	fmt.Printf("  -H \"X-Client-ID: %s\" \\\n", clientID)
	fmt.Printf("  -H \"X-Timestamp: %d\" \\\n", timestamp)
	if withNonce {
		fmt.Printf("  -H \"X-Nonce: %s\" \\\n", nonce)
	}
	fmt.Printf("  -H \"X-Signature: %s\" \\\n", signature)
	fmt.Printf("  \"http://localhost:8080%s\"\n", signPath)

	return nil
}
