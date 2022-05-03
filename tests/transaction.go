package tests

import (
	"time"
)

// TransactionJob is how a transaction load test is parameterized
type TransactionJob struct {
	//TransactionDurationMax the maximum duration for a transaction
	TransactionDurationMax time.Duration `json:"transactionDurationMax,omitempty" yaml:"transactionDurationMax,omitempty"`
	//TransactionDurationMin the minimum duration for a transaction
	TransactionDurationMin time.Duration `json:"transactionDurationMin,omitempty" yaml:"transactionDurationMin,omitempty"`
	//TransactionTimestampSpread the spread (from Now) of the timestamp, generated transactions will have timestamps between
	//`Now` and `Now-TransactionTimestampSpread`
	TransactionTimestampSpread time.Duration `json:"transactionTimestampSpread,omitempty" yaml:"transactionTimestampSpread,omitempty"`
	//MinSpans determines the minimum number of spans generated in a transaction
	MinSpans uint64 `json:"minSpans,omitempty" yaml:"minSpans,omitempty"`
	//MaxSpans determines the maximum number of spans generated in a transaction
	MaxSpans uint64 `json:"maxSpans,omitempty" yaml:"maxSpans,omitempty"`
	//NumReleases determines the maximum number of unique releases generated in a test
	NumReleases uint64 `json:"numReleases,omitempty" yaml:"numReleases,omitempty"`
	//NumUsers determines the maximum number of unique users generated in a test
	NumUsers uint64 `json:"numUsers,omitempty" yaml:"numUsers,omitempty"`
	//Min
	MinBreadcrumbs       uint64   `json:"minBreadcrumbs,omitempty" yaml:"minBreadcrumbs,omitempty"`
	MaxBreadcrumbs       uint64   `json:"maxBreadcrumbs,omitempty" yaml:"maxBreadcrumbs,omitempty"`
	BreadcrumbCategories []string `json:"breadcrumbCategories,omitempty" yaml:"breadcrumbCategories,omitempty"`
	BreadcrumbLevels     []string `json:"breadcrumbLevels,omitempty" yaml:"breadcrumbLevels,omitempty"`
	BreadcrumbsTypes     []string `json:"breadcrumbsTypes,omitempty" yaml:"breadcrumbsTypes,omitempty"`
	BreadcrumbMessages   []string `json:"breadcrumbMessages,omitempty" yaml:"breadcrumbMessages,omitempty"`
	Measurements         []string `json:"measurements,omitempty" yaml:"measurements,omitempty"`
	Operations           []string `json:"operations,omitempty" yaml:"operations,omitempty"`
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
