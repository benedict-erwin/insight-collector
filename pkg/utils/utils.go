package utils

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/benedict-erwin/insight-collector/config"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

var appLocation *time.Location

// init initializes timezone with UTC as default
func init() {
	// Initialize with UTC as default
	appLocation = time.UTC
}

// InitTimezone initializes the application timezone from config
func InitTimezone() error {
	cfg := config.Get()
	timezone := cfg.App.Timezone

	if timezone == "" {
		logger.Warn().Msg("No timezone configured, using UTC")
		appLocation = time.UTC
		return nil
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		logger.Error().Err(err).Str("timezone", timezone).Msg("Failed to load timezone, using UTC")
		appLocation = time.UTC
		return err
	}

	appLocation = loc
	logger.Info().Str("timezone", timezone).Msg("Timezone initialized")
	return nil
}

// Now returns current time in application timezone
func Now() time.Time {
	return time.Now().In(appLocation)
}

// NowFormatted returns current time formatted in RFC3339 with app timezone
func NowFormatted() string {
	return Now().Format(time.RFC3339)
}

// FormatTime formats given time to application timezone
func FormatTime(t time.Time) string {
	return t.In(appLocation).Format(time.RFC3339)
}

// GetLocation returns the current application location
func GetLocation() *time.Location {
	return appLocation
}

// UcFirstUnicode returns a copy of the input string with the first character uppercased.
// It handles Unicode characters correctly.
func UcFirst(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ClearScreen execute clear screen command
func ClearScreen() {
	time.Sleep(1 * time.Millisecond)
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Windows: cls clears screen but not scrollback
		// Need to use mode command or PowerShell
		cmd = exec.Command("cmd", "/c", "cls")
		cmd.Run()

		// Additional clear for Windows terminal scrollback
		fmt.Print("\033[3J")
		os.Stdout.Sync()
	} else {
		// Linux/macOS: clear with scrollback
		cmd = exec.Command("clear")
		cmd.Env = append(os.Environ(), "TERM=xterm")
		cmd.Stdout = os.Stdout
		cmd.Run()

		// Extra ANSI codes for stubborn terminals
		fmt.Print("\033[3J\033[2J\033[H")
		os.Stdout.Sync()
	}
}

// EncodeForURL encodes string to URL-safe base64
func EncodeForURL(data string) string {
	return base64.URLEncoding.EncodeToString([]byte(data))
}

// DecodeFromURL decodes URL-safe base64 string
func DecodeFromURL(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("invalid base64 encoding: %w", err)
	}
	return string(decoded), nil
}

// CreateRecordID creates a base64-encoded ID from timestamp and request_id
func CreateRecordID(timestamp, requestID string) string {
	combined := timestamp + "|" + requestID
	return EncodeForURL(combined)
}

// ParseRecordID parses base64-encoded ID back to timestamp and request_id
func ParseRecordID(encodedID string) (timestamp, requestID string, err error) {
	decoded, err := DecodeFromURL(encodedID)
	if err != nil {
		return "", "", err
	}

	// Split by delimiter
	parts := strings.Split(decoded, "|")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid record ID format")
	}

	return parts[0], parts[1], nil
}
