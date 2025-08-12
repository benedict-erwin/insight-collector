package v2oss

// PaginationRequest represents cursor-based pagination request  
type PaginationRequest struct {
	Length    int              `json:"length" validate:"required,min=1,max=100"`
	Cursor    *string          `json:"cursor,omitempty"`    // RFC3339 timestamp string for cursor-based pagination
	Direction string           `json:"direction" validate:"required,oneof=next prev"`
	Filters   []FilterItem     `json:"filters"`
	Range     *DateRangeFilter `json:"range,omitempty"`
}

// FilterItem represents individual filter criteria
type FilterItem struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
}

// DateRangeFilter represents date range filtering
type DateRangeFilter struct {
	Start string `json:"start,omitempty"` // YYYY-MM-DD format
	End   string `json:"end,omitempty"`   // YYYY-MM-DD format
}

// PaginationResponse represents the response structure with pagination info
type PaginationResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

// PaginationInfo contains cursor-based pagination metadata
type PaginationInfo struct {
	Length     int     `json:"length"`
	HasNext    bool    `json:"has_next"`
	HasPrev    bool    `json:"has_prev"`
	NextCursor *string `json:"next_cursor,omitempty"` // RFC3339 timestamp for next page
	PrevCursor *string `json:"prev_cursor,omitempty"` // RFC3339 timestamp for prev page  
	Direction  string  `json:"direction"`
	Total      int     `json:"total,omitempty"`       // Optional total count
}

// QueryBuilderConfig configuration for query builder
type QueryBuilderConfig struct {
	Measurement string          `json:"measurement"`
	ValidTags   map[string]bool `json:"valid_tags"`   // Tag fields that can be filtered
	ValidFields map[string]bool `json:"valid_fields"` // Field columns that can be filtered
	Columns     []string        `json:"columns"`      // Columns to select in result
	CountField  string          `json:"count_field"`  // Field to use for counting unique records (optional)
}