package auth

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"strconv"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/utils"
)

// Global variable for windowTime
var windowTime int64 = 30

// SignaturePayload represents the data structure for signature generation
type SignaturePayload struct {
	ClientID  string `json:"client_id"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce,omitempty"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Body      string `json:"body,omitempty"`
}

// ToSignatureString converts payload to canonical JSON string for signing
func (p SignaturePayload) ToSignatureString() string {
	// Canonical JSON ensures consistent ordering
	data, _ := json.Marshal(p)
	return string(data)
}

// getHashFunction returns hash function based on config algorithm
func getHashFunction() (func() hash.Hash, crypto.Hash, error) {
	algorithm := config.Get().Auth.Algorithm
	switch algorithm {
	case "RS256", "HS256":
		return sha256.New, crypto.SHA256, nil
	case "RS512", "HS512":
		return sha512.New, crypto.SHA512, nil
	default:
		return nil, 0, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

// GenerateRSASignature generates RSA signature for the payload
func GenerateRSASignature(payload SignaturePayload, privateKey *rsa.PrivateKey) (string, error) {
	payloadStr := payload.ToSignatureString()
	hashFunc, cryptoHash, err := getHashFunction()
	if err != nil {
		return "", fmt.Errorf("failed to get hash function: %v", err)
	}

	h := hashFunc()
	h.Write([]byte(payloadStr))
	hashed := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, cryptoHash, hashed)
	if err != nil {
		return "", fmt.Errorf("failed to generate RSA signature: %v", err)
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// GenerateHMACSignature generates HMAC signature for the payload
func GenerateHMACSignature(payload SignaturePayload, secretKey string) (string, error) {
	payloadStr := payload.ToSignatureString()
	hashFunc, _, err := getHashFunction()
	if err != nil {
		return "", fmt.Errorf("failed to get hash function: %v", err)
	}

	mac := hmac.New(hashFunc, []byte(secretKey))
	mac.Write([]byte(payloadStr))
	signature := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(signature), nil
}

// VerifySignature verifies the request signature and returns client config
func VerifySignature(clientID, timestampStr, nonce, method, path, body, signatureStr string) (*config.ClientConfig, error) {
	// Parse timestamp
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		logger.Warn().
			Str("client_id", clientID).
			Str("timestamp", timestampStr).
			Msg("Invalid timestamp format")
		return nil, fmt.Errorf("invalid timestamp format")
	}

	// Check timestamp validity (5 minutes window)
	now := utils.Now().Unix()
	if now-timestamp > windowTime || timestamp-now > windowTime {
		logger.Warn().
			Str("client_id", clientID).
			Int64("timestamp", timestamp).
			Int64("current_time", now).
			Int64("diff", now-timestamp).
			Msg("Request timestamp expired or too far in future")
		return nil, fmt.Errorf("request timestamp expired")
	}

	// Get client info
	_, clientConfig, exists := GetClientInfo(clientID)
	if !exists {
		logger.Warn().
			Str("client_id", clientID).
			Msg("Unknown client ID in signature verification")
		return nil, fmt.Errorf("unknown client_id: %s", clientID)
	}

	if !clientConfig.Active {
		logger.Warn().
			Str("client_id", clientID).
			Str("client_name", clientConfig.ClientName).
			Msg("Inactive client attempted signature verification")
		return nil, fmt.Errorf("client %s (%s) is inactive", clientID, clientConfig.ClientName)
	}

	// Optional nonce checking for replay attack prevention
	if nonce != "" {
		nonceMutex.Lock()
		if usedTimestamp, exists := usedNonces[nonce]; exists {
			nonceMutex.Unlock()
			logger.Warn().
				Str("client_id", clientID).
				Str("nonce", nonce).
				Int64("previous_timestamp", usedTimestamp).
				Int64("current_timestamp", timestamp).
				Msg("Nonce replay attack detected")
			return nil, fmt.Errorf("nonce already used (replay attack)")
		}
		// Store nonce with current timestamp
		usedNonces[nonce] = timestamp
		nonceMutex.Unlock()
		
		logger.Debug().
			Str("client_id", clientID).
			Str("nonce", nonce).
			Msg("Nonce registered successfully")
	}

	// Build payload for verification
	payload := SignaturePayload{
		ClientID:  clientID,
		Timestamp: timestamp,
		Nonce:     nonce,
		Method:    method,
		Path:      path,
		Body:      body,
	}

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(signatureStr)
	if err != nil {
		logger.Warn().
			Str("client_id", clientID).
			Err(err).
			Msg("Failed to decode signature")
		return nil, fmt.Errorf("invalid signature format")
	}

	// Verify signature based on client auth type
	switch clientConfig.AuthType {
	case "rsa":
		err = verifyRSASignature(payload, signature, clientID)
	case "hmac":
		err = verifyHMACSignature(payload, signature, clientID)
	default:
		logger.Error().
			Str("client_id", clientID).
			Str("auth_type", clientConfig.AuthType).
			Msg("Invalid auth type for signature verification")
		return nil, fmt.Errorf("invalid auth_type: %s", clientConfig.AuthType)
	}

	if err != nil {
		logger.Warn().
			Str("client_id", clientID).
			Str("client_name", clientConfig.ClientName).
			Str("auth_type", clientConfig.AuthType).
			Str("algorithm", config.Get().Auth.Algorithm).
			Err(err).
			Msg("Signature verification failed")
		return nil, fmt.Errorf("signature verification failed: %v", err)
	}

	logger.Info().
		Str("client_id", clientID).
		Str("client_name", clientConfig.ClientName).
		Str("auth_type", clientConfig.AuthType).
		Str("algorithm", config.Get().Auth.Algorithm).
		Str("method", method).
		Str("path", path).
		Int64("timestamp", timestamp).
		Msg("Signature verification successful")

	return &clientConfig, nil
}

// verifyRSASignature verifies RSA signature
func verifyRSASignature(payload SignaturePayload, signature []byte, clientID string) error {
	publicKey, _, exists := GetClientInfo(clientID)
	if !exists || publicKey == nil {
		return fmt.Errorf("RSA public key not found for client: %s", clientID)
	}

	hashFunc, cryptoHash, err := getHashFunction()
	if err != nil {
		return fmt.Errorf("failed to get hash function: %v", err)
	}

	payloadStr := payload.ToSignatureString()
	h := hashFunc()
	h.Write([]byte(payloadStr))
	hashed := h.Sum(nil)

	return rsa.VerifyPKCS1v15(publicKey, cryptoHash, hashed, signature)
}

// verifyHMACSignature verifies HMAC signature
func verifyHMACSignature(payload SignaturePayload, signature []byte, clientID string) error {
	secretKey, exists := GetClientSecretKey(clientID)
	if !exists {
		return fmt.Errorf("HMAC secret key not found for client: %s", clientID)
	}

	hashFunc, _, err := getHashFunction()
	if err != nil {
		return fmt.Errorf("failed to get hash function: %v", err)
	}

	payloadStr := payload.ToSignatureString()
	mac := hmac.New(hashFunc, []byte(secretKey))
	mac.Write([]byte(payloadStr))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return fmt.Errorf("HMAC signature mismatch")
	}

	return nil
}

// GenerateSignature generates signature for HMAC clients (helper function for CLI)
func GenerateSignature(payload SignaturePayload, secretKey string) (string, error) {
	return GenerateHMACSignature(payload, secretKey)
}
