package v2oss

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// QueryBuilder builds InfluxDB Flux queries dynamically based on pagination request
type QueryBuilder struct {
	config QueryBuilderConfig
}

// NewQueryBuilder creates a new query builder instance with configuration
func NewQueryBuilder(config QueryBuilderConfig) *QueryBuilder {
	return &QueryBuilder{
		config: config,
	}
}

// BuildQuery constructs cursor-based Flux query for true server-side pagination
func (qb *QueryBuilder) BuildQuery(req *PaginationRequest, bucket string) (string, error) {
	if err := qb.ValidateRequest(req); err != nil {
		return "", err
	}

	// Validate bucket parameter
	if bucket == "" {
		return "", fmt.Errorf("bucket parameter is required")
	}

	// Build time range filter
	timeRange, err := qb.buildTimeRange(req.Range)
	if err != nil {
		return "", fmt.Errorf("invalid time range: %w", err)
	}

	// Build cursor filter
	cursorFilter := qb.buildCursorFilter(req.Cursor, req.Direction)

	// Build dynamic filters
	filters := qb.buildFilters(req.Filters)

	// Build columns selection
	columns := qb.buildColumns()

	// Calculate server-side safety limit
	var safetyLimit string
	if req.Cursor == nil || *req.Cursor == "" {
		// Page 1: Apply safety limit to prevent catastrophic data transfer
		limit := req.Length * 10 // 10x safety margin for first page
		if limit < 50 {
			limit = 50 // Minimum safety buffer
		}
		if limit > 1000 {
			limit = 1000 // Maximum safety cap
		}
		safetyLimit = fmt.Sprintf("\n  |> limit(n: %d)", limit)
	} else {
		// Page 2+: No safety limit needed (cursor filtering is efficient)
		safetyLimit = ""
	}

	// Build complete query with safety limit for Page 1
	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(%s)
  |> filter(fn: (r) => r["_measurement"] == "%s")%s%s
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")%s
  |> sort(columns: ["_time"], desc: %t)%s`,
		bucket, // Use provided bucket parameter
		timeRange,
		qb.config.Measurement,
		filters,
		cursorFilter,            // Cursor filtering for Page 2+
		columns,                 // Keep columns after pivot
		req.Direction == "next", // Sort direction
		safetyLimit,             // Safety limit for Page 1 only
	)

	return strings.TrimSpace(query), nil
}

// BuildCountQuery constructs query to get total count of records (ignores cursor for total count)
func (qb *QueryBuilder) BuildCountQuery(req *PaginationRequest, bucket string) (string, error) {
	// Validate bucket parameter
	if bucket == "" {
		return "", fmt.Errorf("bucket parameter is required")
	}

	// Build time range filter
	timeRange, err := qb.buildTimeRange(req.Range)
	if err != nil {
		return "", fmt.Errorf("invalid time range: %w", err)
	}

	// Build dynamic filters (no cursor for total count)
	filters := qb.buildFilters(req.Filters)

	// Simple count query using CountField
	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(%s)
  |> filter(fn: (r) => r["_measurement"] == "%s")
  |> filter(fn: (r) => r["_field"] == "%s")%s
  |> count()
  |> group()
  |> sum()`,
		bucket, // Use provided bucket parameter
		timeRange,
		qb.config.Measurement,
		qb.config.CountField,
		filters,
	)

	return strings.TrimSpace(query), nil
}

// buildTimeRange constructs the time range filter based on start/end dates
func (qb *QueryBuilder) buildTimeRange(dateRange *DateRangeFilter) (string, error) {
	if dateRange == nil {
		// Default to last 7 days if no range specified
		return "start: -7d", nil
	}

	// Parse dates
	var startTime, endTime time.Time
	var err error

	if dateRange.Start != "" {
		startTime, err = time.Parse("2006-01-02", dateRange.Start)
		if err != nil {
			return "", fmt.Errorf("invalid start date format, expected YYYY-MM-DD: %w", err)
		}
	}

	if dateRange.End != "" {
		endTime, err = time.Parse("2006-01-02", dateRange.End)
		if err != nil {
			return "", fmt.Errorf("invalid end date format, expected YYYY-MM-DD: %w", err)
		}
		// Set end time to end of day
		endTime = endTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Build time range based on available dates
	if dateRange.Start != "" && dateRange.End != "" {
		// Both start and end provided - range query
		if startTime.After(endTime) {
			return "", fmt.Errorf("start date cannot be after end date")
		}
		return fmt.Sprintf(`start: %s, stop: %s`,
			startTime.Format("2006-01-02T15:04:05Z"),
			endTime.Format("2006-01-02T15:04:05Z")), nil
	} else if dateRange.Start != "" {
		// Only start date - exact day query
		endOfDay := startTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return fmt.Sprintf(`start: %s, stop: %s`,
			startTime.Format("2006-01-02T15:04:05Z"),
			endOfDay.Format("2006-01-02T15:04:05Z")), nil
	} else if dateRange.End != "" {
		// Only end date - exact day query
		startOfDay := endTime.Add(-23*time.Hour - 59*time.Minute - 59*time.Second)
		return fmt.Sprintf(`start: %s, stop: %s`,
			startOfDay.Format("2006-01-02T15:04:05Z"),
			endTime.Format("2006-01-02T15:04:05Z")), nil
	}

	// Fallback to default range
	return "start: -7d", nil
}

// buildFilters constructs dynamic filter conditions based on provided filters
func (qb *QueryBuilder) buildFilters(filters []FilterItem) string {
	if len(filters) == 0 {
		return ""
	}

	var filterConditions []string

	for _, filter := range filters {
		key := strings.ToLower(strings.TrimSpace(filter.Key))
		value := strings.TrimSpace(filter.Value)

		if value == "" {
			continue // Skip empty values
		}

		// Escape quotes in filter values
		escapedValue := strings.ReplaceAll(value, `"`, `\"`)

		if qb.config.ValidTags[key] {
			// Tag-based filter (exact match)
			filterConditions = append(filterConditions,
				fmt.Sprintf(`filter(fn: (r) => r["%s"] == "%s")`, key, escapedValue))
		} else if qb.config.ValidFields[key] {
			// Field-based filter (exact match for strings)
			filterConditions = append(filterConditions,
				fmt.Sprintf(`filter(fn: (r) => r["%s"] == "%s")`, key, escapedValue))
		}
		// Invalid keys are silently ignored for security
	}

	if len(filterConditions) == 0 {
		return ""
	}

	// Join all filters with pipe operators
	return "\n  |> " + strings.Join(filterConditions, "\n  |> ")
}

// buildColumns constructs column selection based on configuration
func (qb *QueryBuilder) buildColumns() string {
	if len(qb.config.Columns) == 0 {
		return "" // No column filtering, return all
	}

	// Build columns list with proper quoting
	var quotedColumns []string
	for _, col := range qb.config.Columns {
		quotedColumns = append(quotedColumns, fmt.Sprintf(`"%s"`, col))
	}

	return fmt.Sprintf("\n  |> keep(columns: [%s])", strings.Join(quotedColumns, ", "))
}

// buildCursorFilter constructs cursor-based time filtering for pagination
func (qb *QueryBuilder) buildCursorFilter(cursor *string, direction string) string {
	if cursor == nil || *cursor == "" {
		return "" // No cursor filter for first page
	}

	// Parse cursor timestamp
	cursorTime := *cursor

	// Build cursor filter based on direction
	var operator string
	if direction == "next" {
		operator = "<" // Get records older than cursor (next page)
	} else {
		operator = ">" // Get records newer than cursor (prev page)
	}

	return fmt.Sprintf("\n  |> filter(fn: (r) => r._time %s time(v: \"%s\"))", operator, cursorTime)
}

// ValidateRequest validates the cursor-based pagination request structure
func (qb *QueryBuilder) ValidateRequest(req *PaginationRequest) error {
	if req.Length <= 0 {
		return fmt.Errorf("length must be greater than 0")
	}

	if req.Length > 100 {
		return fmt.Errorf("length cannot exceed 100 records per request")
	}

	// Validate direction
	if req.Direction != "next" && req.Direction != "prev" {
		return fmt.Errorf("direction must be 'next' or 'prev'")
	}

	// Validate cursor format if provided
	if req.Cursor != nil && *req.Cursor != "" {
		if _, err := time.Parse(time.RFC3339, *req.Cursor); err != nil {
			return fmt.Errorf("invalid cursor format, expected RFC3339 timestamp: %w", err)
		}
	}

	// Validate date range format if provided
	if req.Range != nil {
		if req.Range.Start != "" {
			if _, err := time.Parse("2006-01-02", req.Range.Start); err != nil {
				return fmt.Errorf("invalid start date format, expected YYYY-MM-DD")
			}
		}
		if req.Range.End != "" {
			if _, err := time.Parse("2006-01-02", req.Range.End); err != nil {
				return fmt.Errorf("invalid end date format, expected YYYY-MM-DD")
			}
		}
	}

	return nil
}

// GetPaginationInfo calculates cursor-based pagination metadata
func (qb *QueryBuilder) GetPaginationInfo(req *PaginationRequest, results []map[string]interface{}, totalRecords int) PaginationInfo {
	var nextCursor, prevCursor *string

	// Generate cursors from results
	if len(results) > 0 {
		// Extract timestamp as string helper function
		extractTimestamp := func(record map[string]interface{}) *string {
			if timeVal, ok := record["_time"]; ok {
				switch v := timeVal.(type) {
				case string:
					return &v
				case time.Time:
					timeStr := v.Format(time.RFC3339)
					return &timeStr
				default:
					timeStr := fmt.Sprintf("%v", v)
					if timeStr != "" {
						return &timeStr
					}
				}
			}
			return nil
		}

		// Next cursor (last record timestamp)
		nextCursor = extractTimestamp(results[len(results)-1])

		// Previous cursor (first record timestamp)
		prevCursor = extractTimestamp(results[0])
	}

	// Determine if there are more pages
	hasNext := len(results) == req.Length             // If we got full page, likely more exists
	hasPrev := req.Cursor != nil && *req.Cursor != "" // If we have cursor, prev exists

	return PaginationInfo{
		Length:     req.Length,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		Direction:  req.Direction,
		Total:      totalRecords,
	}
}

// GetTotalCount executes count query and returns total records using provided client
func (qb *QueryBuilder) GetTotalCount(req *PaginationRequest, client *Client) int {
	bucket := client.config.Bucket
	countQuery, err := qb.BuildCountQuery(req, bucket)
	if err != nil {
		return 0
	}

	result, err := client.Query(countQuery)
	if err != nil {
		return 0
	}
	countIterator, ok := result.(*QueryIterator)
	if !ok {
		return 0
	}
	if countIterator != nil {
		defer func() { _ = countIterator.Close() }()

		// Parse count result
		for countIterator.Next() {
			record := countIterator.Record()
			if record != nil && record["_value"] != nil {
				switch v := record["_value"].(type) {
				case int64:
					return int(v)
				case float64:
					return int(v)
				case string:
					if parsed, parseErr := strconv.Atoi(v); parseErr == nil {
						return parsed
					}
				}
			}
		}
	}
	return 0
}

// ExecuteDataQuery builds and executes the main data query with client-side limiting (reusable)
func (qb *QueryBuilder) ExecuteDataQuery(req *PaginationRequest, client *Client) ([]map[string]interface{}, error) {
	// Build query
	bucket := client.config.Bucket
	query, err := qb.BuildQuery(req, bucket)
	if err != nil {
		return nil, err
	}

	// Execute query
	result, err := client.Query(query)
	if err != nil {
		return nil, err
	}

	iterator, ok := result.(*QueryIterator)
	if !ok || iterator == nil {
		return []map[string]interface{}{}, nil
	}

	defer func() { _ = iterator.Close() }()

	// Parse results
	var results []map[string]interface{}
	for iterator.Next() {
		record := iterator.Record()
		if record != nil {
			// Filter out internal InfluxDB fields
			cleanRecord := make(map[string]interface{})
			for key, value := range record {
				if key != "result" && key != "table" && key != "_start" && key != "_stop" {
					cleanRecord[key] = value
				}
			}
			results = append(results, cleanRecord)
		}
	}

	// Check for iterator errors
	if err := iterator.Err(); err != nil {
		return nil, err
	}

	// Apply client-side limiting
	if len(results) > req.Length {
		results = results[:req.Length]
	}

	return results, nil
}

// GetByTimestampAndUniqueID retrieves a single record by timestamp and unique column (reusable method)
func (qb *QueryBuilder) GetByTimestampAndUniqueID(timestamp, columnKey string, columnValue string, client *Client) (map[string]interface{}, error) {
	bucket := client.config.Bucket
	if timestamp == "" {
		return nil, fmt.Errorf("timestamp cannot be empty")
	}
	if columnKey == "" {
		return nil, fmt.Errorf("column_key cannot be empty")
	}
	if columnValue == "" {
		return nil, fmt.Errorf("unique_id cannot be empty")
	}

	// Validate bucket parameter
	if bucket == "" {
		return nil, fmt.Errorf("bucket parameter is required")
	}

	// Validate timestamp format
	parsedTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format, expected RFC3339: %w", err)
	}

	// Build time window (±1 minute around the timestamp for efficiency)
	startTime := parsedTime.Add(-1 * time.Minute)
	endTime := parsedTime.Add(1 * time.Minute)

	// Build query with time window and <column_key> filter
	// Fix: Filter <column_key> AFTER pivot since <column_key> becomes a column after pivot
	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => r["_measurement"] == "%s")
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
  |> filter(fn: (r) => r["%s"] == "%s")
  |> limit(n: 1)`,
		bucket, // Use provided bucket parameter
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		qb.config.Measurement,
		columnKey,
		strings.ReplaceAll(columnValue, `"`, `\"`), // Escape quotes
	)

	// Execute query
	result, err := client.Query(query)
	if err != nil {
		// Add debug info for failed queries
		return nil, fmt.Errorf("failed to execute query for timestamp=%s, %s=%s: %w", timestamp, columnKey, columnValue, err)
	}

	iterator, ok := result.(*QueryIterator)
	if !ok || iterator == nil {
		return nil, fmt.Errorf("no record found with timestamp: %s and %s: %s", timestamp, columnKey, columnValue)
	}

	defer func() { _ = iterator.Close() }()

	// Parse single result
	for iterator.Next() {
		record := iterator.Record()
		if record != nil {
			// Filter out internal InfluxDB fields
			cleanRecord := make(map[string]interface{})
			for key, value := range record {
				if key != "result" && key != "table" && key != "_start" && key != "_stop" {
					cleanRecord[key] = value
				}
			}
			return cleanRecord, nil
		}
	}

	// Check for iterator errors
	if err := iterator.Err(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return nil, fmt.Errorf("no record found with timestamp: %s and %s: %s", timestamp, columnKey, columnValue)
}
