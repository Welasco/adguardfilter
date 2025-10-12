package model

// Schedule represents the scheduling configuration
type Schedule struct {
	TimeZone string `json:"time_zone"`
}

// ServiceConfig represents the service configuration with IDs and schedule
type ServiceConfig struct {
	IDs      []string `json:"ids"`
	Schedule Schedule `json:"schedule"`
}

// ResetServiceMinConfig represents a temporary service configuration with a reset timer
type ResetServiceMinConfig struct {
	ServiceConfig ServiceConfig `json:"config"`
	ResetAfterMin int           `json:"reset_after_min"` // Duration in minutes before resetting to default
}

// ResetServiceDateTimeConfig represents a temporary service configuration with a reset deadline
type ResetServiceDateTimeConfig struct {
	ServiceConfig ServiceConfig `json:"config"`
	ResetDateTime string        `json:"reset_date_time"` // ISO 8601 datetime string (e.g., "2025-10-12T15:30:00Z")
}
