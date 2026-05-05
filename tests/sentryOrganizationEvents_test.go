package tests

import (
	"encoding/json"
	"net/url"
	"testing"

	vegeta "github.com/tsenart/vegeta/lib"
)

func TestSentryOrganizationEventsUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "my-org",
		"fieldSets": [["title", "count()"], ["transaction", "p95(transaction.duration)"]],
		"datasets": ["discover", "transactions"],
		"queries": ["", "event.type:transaction"],
		"sortBy": ["-count()"]
	}`)

	lt := newSentryOrganizationEventsLoadTester("https://sentry.io", raw)
	slt, ok := lt.(*sentryOrganizationEventsLoadTester)
	if !ok {
		t.Fatal("unexpected type")
	}
	if len(slt.params.FieldSets) != 2 {
		t.Errorf("expected 2 field sets, got %d", len(slt.params.FieldSets))
	}
	if len(slt.params.Datasets) != 2 {
		t.Errorf("expected 2 datasets, got %d", len(slt.params.Datasets))
	}
}

func TestSentryOrganizationEventsTargeter(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org",
		"fieldSets": [["title", "count()", "count_unique(user)"]],
		"datasets": ["discover"],
		"statsPeriods": ["24h"],
		"limits": [50],
		"sortBy": ["-count()"]
	}`)

	lt := newSentryOrganizationEventsLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	tgt := &vegeta.Target{}
	if err := targeter(tgt); err != nil {
		t.Fatalf("targeter error: %v", err)
	}

	if tgt.Method != "GET" {
		t.Errorf("expected GET, got %s", tgt.Method)
	}
	if tgt.Header.Get("Authorization") != "Bearer sntrys_test" {
		t.Errorf("unexpected auth header: %s", tgt.Header.Get("Authorization"))
	}

	parsed, err := url.Parse(tgt.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	if parsed.Path != "/api/0/organizations/test-org/events/" {
		t.Errorf("unexpected path: %s", parsed.Path)
	}

	fields := parsed.Query()["field"]
	if len(fields) != 3 {
		t.Errorf("expected 3 field params, got %d: %v", len(fields), fields)
	}
	if parsed.Query().Get("dataset") != "discover" {
		t.Errorf("unexpected dataset: %s", parsed.Query().Get("dataset"))
	}
	if parsed.Query().Get("per_page") != "50" {
		t.Errorf("unexpected per_page: %s", parsed.Query().Get("per_page"))
	}
	if parsed.Query().Get("sort") != "-count()" {
		t.Errorf("unexpected sort: %s", parsed.Query().Get("sort"))
	}
}

func TestSentryOrganizationEventsTargeterDefaults(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryOrganizationEventsLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	tgt := &vegeta.Target{}
	if err := targeter(tgt); err != nil {
		t.Fatalf("targeter error: %v", err)
	}

	parsed, _ := url.Parse(tgt.URL)
	fields := parsed.Query()["field"]
	if len(fields) != 2 {
		t.Errorf("expected 2 default field params, got %d: %v", len(fields), fields)
	}
	if parsed.Query().Get("statsPeriod") != "24h" {
		t.Errorf("expected default statsPeriod '24h', got %s", parsed.Query().Get("statsPeriod"))
	}
	if parsed.Query().Get("dataset") != "" {
		t.Errorf("expected no dataset param when empty, got %s", parsed.Query().Get("dataset"))
	}
}

func TestSentryOrganizationEventsTargeterNilTarget(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryOrganizationEventsLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	if err := targeter(nil); err != vegeta.ErrNilTarget {
		t.Errorf("expected ErrNilTarget, got %v", err)
	}
}
