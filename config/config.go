package config

import "time"

const (
	// How long the printer needs to have been active to honor the shutdown option.
	MinRunTimeForShutdown = 3 * time.Minute
	// Execute emergency shutdown if printer stalled for this duration.
	EmergencyStallTimeout = 5 * time.Minute
)
