package tests

import (
	"time"
)

// TransactionJob is how a transaction load test is parameterized
type TransactionJob struct {
	TransactionDurationMax     time.Duration
	TransactionDurationMin     time.Duration
	TransactionTimestampSpread time.Duration
	MinSpans                   uint64
	MaxSpans                   uint64
	NumReleases                uint64
	NumUsers                   uint64
	MinBreadCrumbs             uint64
	MaxBreadCrumbs             uint64
	BreadcrumbCategories       []string
	BreadcrumbLevels           []string
	BreadcrumbsTypes           []string
	BreadcrumbMessages         []string
	Measurements               []string
	Operations                 []string
}

// Transaction defines the JSON format of a Sentry transaction,
// NOTE: this is just part of a Sentry Event, if we need to emit
// other Events convert this structure into an Event struct and
// add the other fields to it .
type Transaction struct {
	Timestamp      string             `json:"timestamp,omitempty"`       //RFC 3339
	StartTimestamp string             `json:"start_timestamp,omitempty"` //RFC 3339
	EventId        string             `json:"event_id"`
	Release        string             `json:"release,omitempty"`
	Transaction    string             `json:"transaction,omitempty"`
	Logger         string             `json:"logger,omitempty"`
	Environment    string             `json:"environment,omitempty"`
	User           User               `json:"user,omitempty"`
	Contexts       Contexts           `json:"contexts,omitempty"`
	Breadcrumbs    []Breadcrumb       `json:"breadcrumbs,omitempty"`
	Measurements   map[string]float64 `json:"measurements,omitempty"`
	Spans          []Span             `json:"spans,omitempty"`
}
