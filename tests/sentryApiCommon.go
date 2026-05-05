package tests

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	vegeta "github.com/tsenart/vegeta/lib"
)

// Fields shared across all 'read' tests
type SentryReadAPICommon struct {
	AuthToken        string   `json:"authToken"`
	OrganizationSlug string   `json:"organizationSlug"`
	ProjectIds       []string `json:"projectIds,omitempty"`
	StatsPeriods     []string `json:"statsPeriods,omitempty"`
	Limits           []int    `json:"limits,omitempty"`
}

func sentryReadAuthHeader(common SentryReadAPICommon) string {
	return fmt.Sprintf("Bearer %s", common.AuthToken)
}

func setSentryReadHeaders(tgt *vegeta.Target, common SentryReadAPICommon) {
	tgt.Header = make(http.Header)
	tgt.Header.Set("Authorization", sentryReadAuthHeader(common))
	tgt.Header.Set("Accept", "application/json")
}

func randomStringChoice(choices []string, fallback string) string {
	if len(choices) == 0 {
		return fallback
	}
	return choices[rand.Intn(len(choices))]
}

func randomIntChoice(choices []int, fallback int) int {
	if len(choices) == 0 {
		return fallback
	}
	return choices[rand.Intn(len(choices))]
}

func randomStringSliceChoice(choices [][]string, fallback []string) []string {
	if len(choices) == 0 {
		return fallback
	}
	return choices[rand.Intn(len(choices))]
}

func randomProjectId(common SentryReadAPICommon) string {
	if len(common.ProjectIds) == 0 {
		return "" // empty project IDs signals "don't add a project filter"
	}
	return common.ProjectIds[rand.Intn(len(common.ProjectIds))]
}

func buildQueryURL(baseURL string, path string, params url.Values) string {
	base := strings.TrimRight(baseURL, "/")
	query := params.Encode()
	if query == "" {
		return fmt.Sprintf("%s%s", base, path)
	}
	return fmt.Sprintf("%s%s?%s", base, path, query)
}
