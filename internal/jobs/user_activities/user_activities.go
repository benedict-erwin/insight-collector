package useractivities

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	uaEntities "github.com/benedict-erwin/insight-collector/internal/entities/user_activities"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/maxmind"
	"github.com/benedict-erwin/insight-collector/pkg/useragent"
)

// Job processor function
func HandleUserActivitiesLogging(ctx context.Context, t *asynq.Task) error {
	var ua uaEntities.UserActivities
	var req uaEntities.UserActivitiesRequest

	// Logger scope
	log := logger.WithScope(TypeUserActivitiesLogging)

	// Unmarshal request payload
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal payload")
		return err
	}

	// Mapping from request to main entity
	ua.UserID = req.UserID
	ua.SessionID = req.SessionID
	ua.ActivityType = req.ActivityType
	ua.Category = req.Category
	ua.Subcategory = req.Subcategory
	ua.Status = req.Status
	ua.Channel = req.Channel
	ua.EndpointGroup = req.EndpointGroup
	ua.Method = req.Method
	ua.RiskLevel = req.RiskLevel
	ua.RequestID = req.RequestID
	ua.TraceID = req.TraceID
	ua.DurationMs = req.DurationMs
	ua.ResponseCode = req.ResponseCode
	ua.RequestSizeBytes = req.RequestSizeBytes
	ua.ResponseSizeBytes = req.ResponseSizeBytes
	ua.IPAddress = req.IPAddress
	ua.UserAgent = req.UserAgent
	ua.AppVersion = req.AppVersion
	ua.ReferrerURL = req.ReferrerURL
	ua.Endpoint = req.Endpoint
	ua.Details = req.Details
	ua.Timestamp = req.Timestamp

	// UserAgent Check
	detector := useragent.NewFastDetector()
	info := detector.Detect(ua.UserAgent)
	ua.Browser = info.Browser
	ua.BrowserVersion = info.BrowserVersion
	ua.DeviceType = info.Type.String()
	ua.IsBot = info.IsBot
	ua.OS = info.OS
	ua.OSVersion = info.OSVersion

	// IP Geolocation Check
	if ua.IPAddress != "" {
		// Get City Info
		geoLoc := maxmind.LookupCityFromString(ua.IPAddress)
		if geoLoc != nil {
			ua.GeoCountry = strings.ToUpper(geoLoc.CountryCode)
			ua.GeoCity = strings.ToLower(geoLoc.City)
			ua.GeoTimezone = geoLoc.Timezone
			ua.GeoPostal = geoLoc.PostalCode

			// Coordinate format: latitude,longitude
			if geoLoc.Latitude != 0 && geoLoc.Longitude != 0 {
				ua.GeoCoordinates = fmt.Sprintf("%.4f,%.4f", geoLoc.Latitude, geoLoc.Longitude)
			}
		}

		// Get ASN Info
		asnInfo := maxmind.LookupASNFromString(ua.IPAddress)
		if asnInfo != nil && asnInfo.Organization != "" {
			ua.GeoISP = asnInfo.Organization
		}
	}

	// point
	point := ua.ToPoint()
	err := influxdb.WritePoint(point)
	if err != nil {
		return err
	}

	log.Info().
		Str("task_id", t.ResultWriter().TaskID()).
		Str("task_type", t.Type()).
		Str("measurements", ua.GetName()).
		Msg("Job completed successfully")

	return nil
}
