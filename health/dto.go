package health

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// Check represents the result of a health check.
//
// Description is a human-readable description of the check.
//
// ObservedUnit is the unit of measurement for the observed value.
//
// ObservedValue is the observed value of the check.
//
// Status is the status of the check.
// It can be one of the following values: "pass", "warn", or "fail".
type Check struct {
	// A human-readable description of the check.
	Description string `json:"description,omitempty"`
	// Unit of measurement for the observed value.
	ObservedUnit string `json:"observedUnit,omitempty"`
	// Observed value of the check.
	ObservedValue any `json:"observedValue"`
	// Status of the check.
	// It can be one of the following values: "pass", "warn", or "fail".
	Status Status `json:"status"`
}

// Checks is a map of check names to their respective details.
type Checks map[string]Check

// Response represents the result of a health check.
//
// Status is the overall status of the application.
// It can be one of the following values: "pass", "warn", or "fail".
//
// Version is the version of the application.
//
// ReleaseID is the release ID of the application.
// It is used to identify the version of the application.
//
// Checks is a map of check names to their respective details.
type Response struct {
	// Overall status of the application.
	// It can be one of the following values: "pass", "warn", or "fail".
	Status Status `json:"status"`
	// Version of the application.
	Version string `json:"version,omitempty"`
	// Release ID of the application.
	// It is used to identify the version of the application.
	ReleaseID int `json:"releaseId,omitempty"`
	// A map of check names to their respective details.
	Checks Checks `json:"checks,omitempty"`
}
