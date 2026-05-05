package tests

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	vegeta "github.com/tsenart/vegeta/lib"
)

func TestSentryGroupDetailsUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "my-org",
		"latestEventWeight": 3,
		"detailWeight": 7
	}`)

	lt := newSentryGroupDetailsLoadTester("https://sentry.io", raw)
	slt, ok := lt.(*sentryGroupDetailsLoadTester)
	if !ok {
		t.Fatal("unexpected type")
	}
	if slt.params.LatestEventWeight != 3 {
		t.Errorf("expected latestEventWeight 3, got %d", slt.params.LatestEventWeight)
	}
	if slt.params.DetailWeight != 7 {
		t.Errorf("expected detailWeight 7, got %d", slt.params.DetailWeight)
	}
}

func TestSentryGroupDetailsDefaultWeights(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "my-org"
	}`)

	lt := newSentryGroupDetailsLoadTester("https://sentry.io", raw)
	slt := lt.(*sentryGroupDetailsLoadTester)
	if slt.params.DetailWeight != 1 {
		t.Errorf("expected default detailWeight 1, got %d", slt.params.DetailWeight)
	}
	if slt.params.FetchLimit != 100 {
		t.Errorf("expected default fetchLimit 100, got %d", slt.params.FetchLimit)
	}
}

func TestSentryGroupDetailsTargeter(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org",
		"detailWeight": 1,
		"latestEventWeight": 0
	}`)

	lt := newSentryGroupDetailsLoadTester("https://sentry.io", raw)
	slt := lt.(*sentryGroupDetailsLoadTester)
	slt.issueIds = []string{"12345"}
	targeter, _ := slt.GetTargeter()

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
	if parsed.Path != "/api/0/organizations/test-org/issues/12345/" {
		t.Errorf("unexpected path: %s", parsed.Path)
	}
}

func TestSentryGroupDetailsTargeterLatestEventOnly(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org",
		"detailWeight": 0,
		"latestEventWeight": 1
	}`)

	lt := newSentryGroupDetailsLoadTester("https://sentry.io", raw)
	slt := lt.(*sentryGroupDetailsLoadTester)
	slt.issueIds = []string{"99999"}
	targeter, _ := slt.GetTargeter()

	tgt := &vegeta.Target{}
	if err := targeter(tgt); err != nil {
		t.Fatalf("targeter error: %v", err)
	}

	parsed, _ := url.Parse(tgt.URL)
	if !strings.HasSuffix(parsed.Path, "/events/latest/") {
		t.Errorf("expected path to end with /events/latest/, got %s", parsed.Path)
	}
}

func TestSentryGroupDetailsTargeterWeightedDistribution(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org",
		"detailWeight": 7,
		"latestEventWeight": 3
	}`)

	lt := newSentryGroupDetailsLoadTester("https://sentry.io", raw)
	slt := lt.(*sentryGroupDetailsLoadTester)
	slt.issueIds = []string{"12345"}
	targeter, _ := slt.GetTargeter()

	detailCount := 0
	latestCount := 0
	iterations := 1000

	for i := 0; i < iterations; i++ {
		tgt := &vegeta.Target{}
		if err := targeter(tgt); err != nil {
			t.Fatalf("targeter error on iteration %d: %v", i, err)
		}
		parsed, _ := url.Parse(tgt.URL)
		if strings.HasSuffix(parsed.Path, "/events/latest/") {
			latestCount++
		} else {
			detailCount++
		}
	}

	latestRatio := float64(latestCount) / float64(iterations)
	if latestRatio < 0.15 || latestRatio > 0.45 {
		t.Errorf("expected ~30%% latest events, got %.1f%% (detail=%d, latest=%d)",
			latestRatio*100, detailCount, latestCount)
	}
}

func TestSentryGroupDetailsTargeterNoIssueIds(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryGroupDetailsLoadTester("https://sentry.io", raw)
	targeter, _ := lt.GetTargeter()

	tgt := &vegeta.Target{}
	err := targeter(tgt)
	if err == nil {
		t.Error("expected error when no issue IDs available")
	}
}

func TestSentryGroupDetailsTargeterNilTarget(t *testing.T) {
	raw := json.RawMessage(`{
		"authToken": "sntrys_test",
		"organizationSlug": "test-org"
	}`)

	lt := newSentryGroupDetailsLoadTester("https://sentry.io", raw)
	slt := lt.(*sentryGroupDetailsLoadTester)
	slt.issueIds = []string{"123"}
	targeter, _ := slt.GetTargeter()

	if err := targeter(nil); err != vegeta.ErrNilTarget {
		t.Errorf("expected ErrNilTarget, got %v", err)
	}
}
