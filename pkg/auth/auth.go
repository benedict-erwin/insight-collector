package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

var (
	clientPublicKeys = make(map[string]*rsa.PublicKey)
	clientSecretKeys = make(map[string]string)
	clientConfigs    = make(map[string]config.ClientConfig)
	authMutex        sync.RWMutex
	
	// Nonce tracking for replay attack prevention
	usedNonces    = make(map[string]int64) // nonce -> timestamp
	nonceMutex    sync.RWMutex
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
)

// InitAuth loads all client keys (RSA public keys or HMAC secrets) into memory
func InitAuth() error {
	authConfig := config.Get().Auth
	if !authConfig.Enabled {
		logger.Info().Msg("Auth disabled in config")
		return nil
	}

	authMutex.Lock()
	defer authMutex.Unlock()

	loadedCount := 0
	for _, clientConfig := range authConfig.Clients {
		if !clientConfig.Active {
			logger.Info().
				Str("client_id", clientConfig.ClientID).
				Str("client_name", clientConfig.ClientName).
				Msg("Skipping inactive client")
			continue
		}

		// Load keys based on auth type
		switch clientConfig.AuthType {
		case "rsa":
			if clientConfig.KeyPath == "" {
				logger.Error().
					Str("client_id", clientConfig.ClientID).
					Str("client_name", clientConfig.ClientName).
					Msg("RSA auth type requires key_path")
				return fmt.Errorf("RSA client %s (%s) missing key_path",
					clientConfig.ClientID, clientConfig.ClientName)
			}

			// Load RSA public key
			keyBytes, err := os.ReadFile(clientConfig.KeyPath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("client_id", clientConfig.ClientID).
					Str("client_name", clientConfig.ClientName).
					Str("key_path", clientConfig.KeyPath).
					Msg("Failed to load RSA public key")
				return fmt.Errorf("failed to load RSA key for client %s (%s): %v",
					clientConfig.ClientID, clientConfig.ClientName, err)
			}

			publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
			if err != nil {
				logger.Error().
					Err(err).
					Str("client_id", clientConfig.ClientID).
					Str("client_name", clientConfig.ClientName).
					Msg("Failed to parse RSA public key")
				return fmt.Errorf("failed to parse RSA key for client %s (%s): %v",
					clientConfig.ClientID, clientConfig.ClientName, err)
			}

			clientPublicKeys[clientConfig.ClientID] = publicKey

		case "hmac":
			if clientConfig.SecretKey == "" {
				logger.Error().
					Str("client_id", clientConfig.ClientID).
					Str("client_name", clientConfig.ClientName).
					Msg("HMAC auth type requires secret_key")
				return fmt.Errorf("HMAC client %s (%s) missing secret_key",
					clientConfig.ClientID, clientConfig.ClientName)
			}

			clientSecretKeys[clientConfig.ClientID] = clientConfig.SecretKey

		default:
			logger.Error().
				Str("client_id", clientConfig.ClientID).
				Str("client_name", clientConfig.ClientName).
				Str("auth_type", clientConfig.AuthType).
				Msg("Invalid auth type")
			return fmt.Errorf("invalid auth_type '%s' for client %s (%s)",
				clientConfig.AuthType, clientConfig.ClientID, clientConfig.ClientName)
		}

		// Cache client config
		clientConfigs[clientConfig.ClientID] = clientConfig
		loadedCount++

		logger.Info().
			Str("client_id", clientConfig.ClientID).
			Str("client_name", clientConfig.ClientName).
			Str("auth_type", clientConfig.AuthType).
			Strs("permissions", clientConfig.Permissions).
			Msg("Client auth loaded successfully")
	}

	logger.Info().
		Int("loaded_clients", loadedCount).
		Int("total_clients", len(authConfig.Clients)).
		Msg("Auth system initialized")

	// Start nonce cleanup worker
	cleanupCtx, cleanupCancel = context.WithCancel(context.Background())
	go cleanupWorker(cleanupCtx)

	return nil
}

// VerifyJWT verifies JWT token and returns claims
func VerifyJWT(tokenString string) (*Claims, error) {
	// Parse token without verification first to get client_id
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// Extract client_id
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Get public key for this client
	publicKey, clientConfig, exists := GetClientInfo(claims.ClientID)
	if !exists {
		return nil, fmt.Errorf("unknown client_id: %s", claims.ClientID)
	}

	if !clientConfig.Active {
		return nil, fmt.Errorf("client %s (%s) is inactive",
			claims.ClientID, clientConfig.ClientName)
	}

	// Verify signature with correct public key
	token, err = jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		expectedAlg := config.Get().Auth.Algorithm
		if token.Method.Alg() != expectedAlg {
			return nil, fmt.Errorf("unexpected signing method: %v, expected: %s",
				token.Header["alg"], expectedAlg)
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token verification failed: %v", err)
	}

	verifiedClaims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return verifiedClaims, nil
}

// GetClientInfo returns public key and config for client
func GetClientInfo(clientID string) (*rsa.PublicKey, config.ClientConfig, bool) {
	authMutex.RLock()
	defer authMutex.RUnlock()

	clientConfig, configExists := clientConfigs[clientID]
	if !configExists {
		return nil, config.ClientConfig{}, false
	}

	// For RSA clients, return public key
	if clientConfig.AuthType == "rsa" {
		publicKey, keyExists := clientPublicKeys[clientID]
		if keyExists {
			return publicKey, clientConfig, true
		}
	}

	// For HMAC clients, public key is not needed (will be nil)
	if clientConfig.AuthType == "hmac" {
		return nil, clientConfig, true
	}

	return nil, config.ClientConfig{}, false
}

// GetClientSecretKey returns HMAC secret key for client
func GetClientSecretKey(clientID string) (string, bool) {
	authMutex.RLock()
	defer authMutex.RUnlock()

	secretKey, exists := clientSecretKeys[clientID]
	return secretKey, exists
}

// AddClient adds new client to memory cache (called by CLI)
func AddClient(clientConfig config.ClientConfig) error {
	authMutex.Lock()
	defer authMutex.Unlock()

	// Load keys based on auth type
	switch clientConfig.AuthType {
	case "rsa":
		if clientConfig.KeyPath == "" {
			return fmt.Errorf("RSA client %s missing key_path", clientConfig.ClientID)
		}

		// Load RSA public key
		keyBytes, err := os.ReadFile(clientConfig.KeyPath)
		if err != nil {
			logger.Error().
				Err(err).
				Str("client_id", clientConfig.ClientID).
				Str("key_path", clientConfig.KeyPath).
				Msg("Failed to load RSA public key for new client")
			return fmt.Errorf("failed to load RSA key: %v", err)
		}

		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
		if err != nil {
			logger.Error().
				Err(err).
				Str("client_id", clientConfig.ClientID).
				Msg("Failed to parse RSA public key for new client")
			return fmt.Errorf("failed to parse RSA key: %v", err)
		}

		clientPublicKeys[clientConfig.ClientID] = publicKey

	case "hmac":
		if clientConfig.SecretKey == "" {
			return fmt.Errorf("HMAC client %s missing secret_key", clientConfig.ClientID)
		}

		clientSecretKeys[clientConfig.ClientID] = clientConfig.SecretKey

	default:
		return fmt.Errorf("invalid auth_type '%s' for client %s", clientConfig.AuthType, clientConfig.ClientID)
	}

	// Cache client config
	clientConfigs[clientConfig.ClientID] = clientConfig

	logger.Info().
		Str("client_id", clientConfig.ClientID).
		Str("client_name", clientConfig.ClientName).
		Str("auth_type", clientConfig.AuthType).
		Strs("permissions", clientConfig.Permissions).
		Msg("Client added to memory cache")

	return nil
}

// UpdateClient updates existing client in memory cache (called by CLI)
func UpdateClient(clientConfig config.ClientConfig) error {
	authMutex.Lock()
	defer authMutex.Unlock()

	// Check if client exists
	_, exists := clientConfigs[clientConfig.ClientID]
	if !exists {
		return fmt.Errorf("client %s not found in cache", clientConfig.ClientID)
	}

	// Update based on auth type
	switch clientConfig.AuthType {
	case "rsa":
		// Update RSA public key if key path changed
		if clientConfig.KeyPath != "" {
			keyBytes, err := os.ReadFile(clientConfig.KeyPath)
			if err != nil {
				return fmt.Errorf("failed to load RSA key: %v", err)
			}

			publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
			if err != nil {
				return fmt.Errorf("failed to parse RSA key: %v", err)
			}

			clientPublicKeys[clientConfig.ClientID] = publicKey
		}

	case "hmac":
		// Update HMAC secret key
		if clientConfig.SecretKey != "" {
			clientSecretKeys[clientConfig.ClientID] = clientConfig.SecretKey
		}
	}

	// Update client config
	clientConfigs[clientConfig.ClientID] = clientConfig

	logger.Info().
		Str("client_id", clientConfig.ClientID).
		Str("client_name", clientConfig.ClientName).
		Str("auth_type", clientConfig.AuthType).
		Bool("active", clientConfig.Active).
		Msg("Client updated in memory cache")

	return nil
}

// RemoveClient removes client from memory cache (called by CLI)
func RemoveClient(clientID string) error {
	authMutex.Lock()
	defer authMutex.Unlock()

	// Check if client exists
	clientConfig, exists := clientConfigs[clientID]
	if !exists {
		return fmt.Errorf("client %s not found in cache", clientID)
	}

	// Remove from appropriate cache
	switch clientConfig.AuthType {
	case "rsa":
		delete(clientPublicKeys, clientID)
	case "hmac":
		delete(clientSecretKeys, clientID)
	}

	// Remove from client configs
	delete(clientConfigs, clientID)

	logger.Info().
		Str("client_id", clientID).
		Str("client_name", clientConfig.ClientName).
		Msg("Client removed from memory cache")

	return nil
}

// cleanExpiredNonces removes expired nonces from memory
func cleanExpiredNonces() {
	nonceMutex.Lock()
	defer nonceMutex.Unlock()
	
	now := utils.Now().Unix()
	count := 0
	
	for nonce, timestamp := range usedNonces {
		// Remove nonces older than 5 minutes (300 seconds)
		if now-timestamp > 300 {
			delete(usedNonces, nonce)
			count++
		}
	}
	
	if count > 0 {
		logger.Debug().
			Int("cleaned_nonces", count).
			Int("remaining_nonces", len(usedNonces)).
			Msg("Cleaned expired nonces")
	}
}

// cleanupWorker runs periodic nonce cleanup
func cleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	logger.Info().Msg("Nonce cleanup worker started")
	
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Nonce cleanup worker stopped")
			return
		case <-ticker.C:
			cleanExpiredNonces()
		}
	}
}

// StopAuth stops the cleanup worker gracefully
func StopAuth() {
	if cleanupCancel != nil {
		cleanupCancel()
		logger.Info().Msg("Auth system stopped")
	}
}

// ReloadAuth reloads all clients from config (fallback function)
func ReloadAuth() error {
	logger.Info().Msg("Reloading authentication system from config")
	
	// Clear existing cache
	authMutex.Lock()
	clientPublicKeys = make(map[string]*rsa.PublicKey)
	clientSecretKeys = make(map[string]string)
	clientConfigs = make(map[string]config.ClientConfig)
	authMutex.Unlock()

	// Reload from config
	return InitAuth()
}
