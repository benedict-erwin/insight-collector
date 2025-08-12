package securityevents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	securityevents "github.com/benedict-erwin/insight-collector/internal/entities/security_events"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/maxmind"
	"github.com/benedict-erwin/insight-collector/pkg/useragent"
)

// Job processor function
func HandleSecurityEventsLogging(ctx context.Context, t *asynq.Task) error {
	var se securityevents.SecurityEvents
	var req securityevents.SecurityEventsRequest

	// Logger scope
	log := logger.WithScope(TypeSecurityEventsLogging)

	// Unmarshal request payload
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal payload")
		return err
	}

	// Mapping from request to main entity
	se.UserID = req.UserID
	se.SessionID = req.SessionID
	se.IdentifierType = req.IdentifierType
	se.EventType = req.EventType
	se.Severity = req.Severity
	se.AuthStage = req.AuthStage
	se.ActionTaken = req.ActionTaken
	se.DetectionMethod = req.DetectionMethod
	se.Channel = req.Channel
	se.EndpointGroup = req.EndpointGroup
	se.Method = req.Method
	se.RequestID = req.RequestID
	se.TraceID = req.TraceID
	se.IdentifierValue = req.IdentifierValue
	se.AttemptCount = req.AttemptCount
	se.RiskScore = req.RiskScore
	se.ConfidenceScore = req.ConfidenceScore
	se.PreviousSuccessTime = req.PreviousSuccessTime
	se.AffectedResource = req.AffectedResource
	se.DurationMs = req.DurationMs
	se.ResponseCode = req.ResponseCode
	se.IPAddress = req.IPAddress
	se.UserAgent = req.UserAgent
	se.AppVersion = req.AppVersion
	se.Endpoint = req.Endpoint
	se.Details = req.Details
	se.Timestamp = req.Timestamp

	// UserAgent Check
	detector := useragent.NewFastDetector()
	info := detector.Detect(se.UserAgent)
	se.Browser = info.Browser
	se.BrowserVersion = info.BrowserVersion
	se.DeviceType = info.Type.String()
	se.IsBot = info.IsBot
	se.OS = info.OS
	se.OSVersion = info.OSVersion

	// IP Geolocation Check
	if se.IPAddress != "" {
		// Get City Info
		geoLoc := maxmind.LookupCityFromString(se.IPAddress)
		if geoLoc != nil {
			se.GeoCountry = strings.ToUpper(geoLoc.CountryCode)
			se.GeoCity = strings.ToLower(geoLoc.City)
			se.GeoTimezone = geoLoc.Timezone
			se.GeoPostal = geoLoc.PostalCode

			// Coordinate format: latitude,longitude
			if geoLoc.Latitude != 0 && geoLoc.Longitude != 0 {
				se.GeoCoordinates = fmt.Sprintf("%.4f,%.4f", geoLoc.Latitude, geoLoc.Longitude)
			}
		}

		// Get ASN Info
		asnInfo := maxmind.LookupASNFromString(se.IPAddress)
		if asnInfo != nil && asnInfo.Organization != "" {
			se.GeoISP = asnInfo.Organization
		}
	}

	// point
	point := se.ToPoint()
	err := influxdb.WritePoint(point)
	if err != nil {
		return err
	}

	log.Info().
		Str("task_id", t.ResultWriter().TaskID()).
		Str("task_type", t.Type()).
		Str("measurements", se.GetName()).
		Msg("Job completed successfully")

	return nil
}
