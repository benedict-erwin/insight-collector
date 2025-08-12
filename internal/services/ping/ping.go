package ping

import (
	"errors"

	"github.com/benedict-erwin/insight-collector/internal/entities/ping"
)

// Ping returns a simple pong response
func Ping() string {
	return "pong"
}

// PingV2 returns a simple pong response for v2 API
func PingV2() string {
	return "pong"
}

// PingPost validates ping request and returns pong response
func PingPost(req ping.PingRequest) (string, error) {
	if req.Action != "ping" {
		return "", errors.New("invalid action, expected 'ping'")
	}
	return "pong", nil
}
