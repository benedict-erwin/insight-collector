package transactionevents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	transactionevents "github.com/benedict-erwin/insight-collector/internal/entities/transaction_events"
	"github.com/benedict-erwin/insight-collector/pkg/influxdb"
	"github.com/benedict-erwin/insight-collector/pkg/logger"
	"github.com/benedict-erwin/insight-collector/pkg/maxmind"
	"github.com/benedict-erwin/insight-collector/pkg/useragent"
)

// Job processor function
func HandleTransactionEventsLogging(ctx context.Context, t *asynq.Task) error {
	var te transactionevents.TransactionEvents
	var req transactionevents.TransactionEventsRequest

	// Logger scope
	log := logger.WithScope(TypeTransactionEventsLogging)

	// Unmarshal request payload
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal payload")
		return err
	}

	// Mapping from request to main entity
	te.UserID = req.UserID
	te.SessionID = req.SessionID
	te.TransactionType = req.TransactionType
	te.Currency = strings.ToUpper(req.Currency)
	te.PaymentMethod = req.PaymentMethod
	te.Status = req.Status
	te.TransactionNature = req.TransactionNature
	te.MerchantCategory = req.MerchantCategory
	te.Channel = req.Channel
	te.RiskLevel = req.RiskLevel
	te.RequestID = req.RequestID
	te.TraceID = req.TraceID
	te.TransactionID = req.TransactionID
	te.ExternalReferenceID = req.ExternalReferenceID
	te.Amount = req.Amount
	te.FeeAmount = req.FeeAmount
	te.NetAmount = req.NetAmount
	te.ExchangeRate = req.ExchangeRate
	te.ProcessingTimeMs = req.ProcessingTimeMs
	te.DurationMs = req.DurationMs
	te.RetryCount = req.RetryCount
	te.ResponseCode = req.ResponseCode
	te.ApprovalRequired = req.ApprovalRequired
	te.ComplianceScore = req.ComplianceScore
	te.MerchantID = req.MerchantID
	te.DestinationAccount = req.DestinationAccount
	te.IPAddress = req.IPAddress
	te.UserAgent = req.UserAgent
	te.AppVersion = req.AppVersion
	te.Endpoint = req.Endpoint
	te.Method = req.Method
	te.Details = req.Details
	te.Timestamp = req.Timestamp

	// UserAgent Check
	detector := useragent.NewFastDetector()
	info := detector.Detect(te.UserAgent)
	te.Browser = info.Browser
	te.BrowserVersion = info.BrowserVersion
	te.DeviceType = info.Type.String()
	te.IsBot = info.IsBot
	te.OS = info.OS
	te.OSVersion = info.OSVersion

	// IP Geolocation Check
	if te.IPAddress != "" {
		// Get City Info
		geoLoc := maxmind.LookupCityFromString(te.IPAddress)
		if geoLoc != nil {
			te.GeoCountry = strings.ToUpper(geoLoc.CountryCode)
			te.GeoCity = strings.ToLower(geoLoc.City)
			te.GeoTimezone = geoLoc.Timezone
			te.GeoPostal = geoLoc.PostalCode

			// Coordinate format: latitude,longitude
			if geoLoc.Latitude != 0 && geoLoc.Longitude != 0 {
				te.GeoCoordinates = fmt.Sprintf("%.4f,%.4f", geoLoc.Latitude, geoLoc.Longitude)
			}
		}

		// Get ASN Info
		asnInfo := maxmind.LookupASNFromString(te.IPAddress)
		if asnInfo != nil && asnInfo.Organization != "" {
			te.GeoISP = asnInfo.Organization
		}
	}

	// point
	point := te.ToPoint()
	err := influxdb.WritePoint(point)
	if err != nil {
		return err
	}

	log.Info().
		Str("task_id", t.ResultWriter().TaskID()).
		Str("task_type", t.Type()).
		Str("measurements", te.GetName()).
		Msg("Job completed successfully")

	return nil
}
