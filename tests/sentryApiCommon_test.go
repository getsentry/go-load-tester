package tests

import (
	"net/url"
	"testing"

	vegeta "github.com/tsenart/vegeta/lib"
)

func TestSentryReadAuthHeader(t *testing.T) {
	common := SentryReadAPICommon{AuthToken: "sntrys_test123"}
	header := sentryReadAuthHeader(common)
	expected := "Bearer sntrys_test123"
	if header != expected {
		t.Errorf("expected %q, got %q", expected, header)
	}
}

func TestSetSentryReadHeaders(t *testing.T) {
	common := SentryReadAPICommon{AuthToken: "sntrys_test123"}
	tgt := &vegeta.Target{}
	setSentryReadHeaders(tgt, common)

	if tgt.Header.Get("Authorization") != "Bearer sntrys_test123" {
		t.Errorf("unexpected Authorization header: %s", tgt.Header.Get("Authorization"))
	}
	if tgt.Header.Get("Accept") != "application/json" {
		t.Errorf("unexpected Accept header: %s", tgt.Header.Get("Accept"))
	}
}

func TestRandomStringChoice(t *testing.T) {
	result := randomStringChoice(nil, "default")
	if result != "default" {
		t.Errorf("expected default, got %q", result)
	}

	result = randomStringChoice([]string{}, "default")
	if result != "default" {
		t.Errorf("expected default, got %q", result)
	}

	result = randomStringChoice([]string{"only"}, "default")
	if result != "only" {
		t.Errorf("expected 'only', got %q", result)
	}
}

func TestRandomIntChoice(t *testing.T) {
	result := randomIntChoice(nil, 25)
	if result != 25 {
		t.Errorf("expected 25, got %d", result)
	}

	result = randomIntChoice([]int{50}, 25)
	if result != 50 {
		t.Errorf("expected 50, got %d", result)
	}
}

func TestRandomStringSliceChoice(t *testing.T) {
	result := randomStringSliceChoice(nil, []string{"a", "b"})
	if len(result) != 2 || result[0] != "a" {
		t.Errorf("expected fallback, got %v", result)
	}

	choices := [][]string{{"x", "y"}}
	result = randomStringSliceChoice(choices, []string{"a"})
	if len(result) != 2 || result[0] != "x" {
		t.Errorf("expected [x y], got %v", result)
	}
}

func TestRandomProjectId(t *testing.T) {
	common := SentryReadAPICommon{}
	result := randomProjectId(common)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}

	common.ProjectIds = []string{"123"}
	result = randomProjectId(common)
	if result != "123" {
		t.Errorf("expected '123', got %q", result)
	}
}

func TestBuildQueryURL(t *testing.T) {
	params := url.Values{}
	result := buildQueryURL("https://sentry.io", "/api/0/test/", params)
	if result != "https://sentry.io/api/0/test/" {
		t.Errorf("unexpected URL: %s", result)
	}

	params.Set("statsPeriod", "24h")
	params.Set("limit", "25")
	result = buildQueryURL("https://sentry.io/", "/api/0/test/", params)
	parsed, err := url.Parse(result)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	if parsed.Host != "sentry.io" {
		t.Errorf("unexpected host: %s", parsed.Host)
	}
	if parsed.Query().Get("statsPeriod") != "24h" {
		t.Errorf("missing statsPeriod param")
	}
	if parsed.Query().Get("limit") != "25" {
		t.Errorf("missing limit param")
	}
}

func TestBuildQueryURLMultiValue(t *testing.T) {
	params := url.Values{}
	params.Add("field", "title")
	params.Add("field", "count()")
	result := buildQueryURL("https://sentry.io", "/api/0/organizations/test-org/events/", params)
	parsed, err := url.Parse(result)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	fields := parsed.Query()["field"]
	if len(fields) != 2 {
		t.Errorf("expected 2 field params, got %d", len(fields))
	}
}

func TestBuildQueryURLTrailingSlash(t *testing.T) {
	result := buildQueryURL("https://sentry.io/", "/api/0/test/", url.Values{})
	if result != "https://sentry.io/api/0/test/" {
		t.Errorf("trailing slash not handled: %s", result)
	}
}
