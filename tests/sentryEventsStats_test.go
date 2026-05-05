package tests

import (
	"encoding/json"
	"net/url"
	"testing"

	vegeta "github.com/tsenart/vegeta/lib"
)

func TestSentryEventsStatsUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "my-org",
		"yAxes": [["count()"], ["count()", "count_unique(user)"]],
		"intervals": ["5m", "1h"],
		"queries": ["", "transaction:/api/*"],
		"datasets": ["discover"]
	}`)

	lt := newSentryEventsStatsLoadTester("https://sentry.io", raw)
	slt, ok := lt.(*sentryEventsStatsLoadTester)
	if !ok {
		t.Fatal("unexpected type")
	}
	if len(slt.params.YAxes) != 2 {
		t.Errorf("expected 2 yAxes sets, got %d", len(slt.params.YAxes))
	}
	if len(slt.params.Intervals) != 2 {
		t.Errorf("expected 2 intervals, got %d", len(slt.params.Intervals))
	}
}

func TestSentryEventsStatsTargeter(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org",
		"projectIds": ["456"],
		"yAxes": [["count()", "count_unique(user)"]],
		"statsPeriods": ["14d"],
		"intervals": ["1h"],
		"datasets": ["discover"],
		"queries": ["transaction:/api/*"]
	}`)

	lt := newSentryEventsStatsLoadTester("https://sentry.io", raw)
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
	if parsed.Path != "/api/0/organizations/test-org/events-stats/" {
		t.Errorf("unexpected path: %s", parsed.Path)
	}

	yAxes := parsed.Query()["yAxis"]
	if len(yAxes) != 2 {
		t.Errorf("expected 2 yAxis params, got %d: %v", len(yAxes), yAxes)
	}
	if parsed.Query().Get("statsPeriod") != "14d" {
		t.Errorf("unexpected statsPeriod: %s", parsed.Query().Get("statsPeriod"))
	}
	if parsed.Query().Get("interval") != "1h" {
		t.Errorf("unexpected interval: %s", parsed.Query().Get("interval"))
	}
	if parsed.Query().Get("dataset") != "discover" {
		t.Errorf("unexpected dataset: %s", parsed.Query().Get("dataset"))
	}
	if parsed.Query().Get("query") != "transaction:/api/*" {
		t.Errorf("unexpected query: %s", parsed.Query().Get("query"))
	}
	if parsed.Query().Get("project") != "456" {
		t.Errorf("unexpected project: %s", parsed.Query().Get("project"))
	}
}

func TestSentryEventsStatsTargeterDefaults(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryEventsStatsLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	tgt := &vegeta.Target{}
	if err := targeter(tgt); err != nil {
		t.Fatalf("targeter error: %v", err)
	}

	parsed, _ := url.Parse(tgt.URL)
	yAxes := parsed.Query()["yAxis"]
	if len(yAxes) != 1 || yAxes[0] != "count()" {
		t.Errorf("expected default yAxis ['count()'], got %v", yAxes)
	}
	if parsed.Query().Get("statsPeriod") != "24h" {
		t.Errorf("expected default statsPeriod '24h', got %s", parsed.Query().Get("statsPeriod"))
	}
	if parsed.Query().Get("interval") != "1h" {
		t.Errorf("expected default interval '1h', got %s", parsed.Query().Get("interval"))
	}
}

func TestSentryEventsStatsTargeterNilTarget(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryEventsStatsLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	if err := targeter(nil); err != vegeta.ErrNilTarget {
		t.Errorf("expected ErrNilTarget, got %v", err)
	}
}
