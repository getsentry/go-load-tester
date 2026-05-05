package tests

import (
	"encoding/json"
	"net/url"
	"testing"

	vegeta "github.com/tsenart/vegeta/lib"
)

func TestSentryOrganizationGroupIndexUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "my-org",
		"statsPeriods": ["24h", "14d"],
		"limits": [25, 50],
		"sortOptions": ["date", "priority"],
		"queries": ["is:unresolved"]
	}`)

	lt := newSentryOrganizationGroupIndexLoadTester("https://sentry.io", raw)
	slt, ok := lt.(*sentryOrganizationGroupIndexLoadTester)
	if !ok {
		t.Fatal("unexpected type")
	}
	if slt.params.OrganizationSlug != "my-org" {
		t.Errorf("unexpected org slug: %s", slt.params.OrganizationSlug)
	}
	if len(slt.params.SortOptions) != 2 {
		t.Errorf("expected 2 sort options, got %d", len(slt.params.SortOptions))
	}
	if len(slt.params.StatsPeriods) != 2 {
		t.Errorf("expected 2 stats periods, got %d", len(slt.params.StatsPeriods))
	}
}

func TestSentryOrganizationGroupIndexTargeter(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org",
		"projectIds": ["123"],
		"statsPeriods": ["24h"],
		"limits": [25],
		"sortOptions": ["date"],
		"queries": ["is:unresolved"]
	}`)

	lt := newSentryOrganizationGroupIndexLoadTester("https://sentry.io", raw)
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
	if parsed.Path != "/api/0/organizations/test-org/issues/" {
		t.Errorf("unexpected path: %s", parsed.Path)
	}
	if parsed.Query().Get("statsPeriod") != "24h" {
		t.Errorf("unexpected statsPeriod: %s", parsed.Query().Get("statsPeriod"))
	}
	if parsed.Query().Get("sort") != "date" {
		t.Errorf("unexpected sort: %s", parsed.Query().Get("sort"))
	}
	if parsed.Query().Get("query") != "is:unresolved" {
		t.Errorf("unexpected query: %s", parsed.Query().Get("query"))
	}
	if parsed.Query().Get("limit") != "25" {
		t.Errorf("unexpected limit: %s", parsed.Query().Get("limit"))
	}
	if parsed.Query().Get("project") != "123" {
		t.Errorf("unexpected project: %s", parsed.Query().Get("project"))
	}
}

func TestSentryOrganizationGroupIndexTargeterDefaults(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryOrganizationGroupIndexLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	tgt := &vegeta.Target{}
	if err := targeter(tgt); err != nil {
		t.Fatalf("targeter error: %v", err)
	}

	parsed, _ := url.Parse(tgt.URL)
	if parsed.Query().Get("statsPeriod") != "24h" {
		t.Errorf("expected default statsPeriod '24h', got %s", parsed.Query().Get("statsPeriod"))
	}
	if parsed.Query().Get("sort") != "date" {
		t.Errorf("expected default sort 'date', got %s", parsed.Query().Get("sort"))
	}
	if parsed.Query().Get("query") != "is:unresolved" {
		t.Errorf("expected default query 'is:unresolved', got %s", parsed.Query().Get("query"))
	}
	if parsed.Query().Get("project") != "" {
		t.Errorf("expected no project param, got %s", parsed.Query().Get("project"))
	}
}

func TestSentryOrganizationGroupIndexTargeterNilTarget(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryOrganizationGroupIndexLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	if err := targeter(nil); err != vegeta.ErrNilTarget {
		t.Errorf("expected ErrNilTarget, got %v", err)
	}
}
