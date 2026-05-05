package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
)

type SentryIssueDetailJob struct {
	SentryReadAPICommon `json:",inline"`
	FetchLimit          int `json:"fetchLimit,omitempty"`
	LatestEventWeight   int `json:"latestEventWeight,omitempty"`
	DetailWeight        int `json:"detailWeight,omitempty"`
}

type sentryIssueDetailLoadTester struct {
	url      string
	params   SentryIssueDetailJob
	issueIds []string
}

func newSentryIssueDetailLoadTester(targetUrl string, rawParams json.RawMessage) LoadTester {
	var params SentryIssueDetailJob
	if err := json.Unmarshal(rawParams, &params); err != nil {
		log.Error().Err(err).Msgf("invalid sentryIssueDetail params received\nraw data\n%s", rawParams)
	}

	if params.FetchLimit == 0 {
		params.FetchLimit = 100
	}
	if params.DetailWeight == 0 && params.LatestEventWeight == 0 {
		params.DetailWeight = 1
	}

	// Utility function that lets us not hardcode issue IDs into every test we want to run;
	// instead we just do a single GET to the issue stream endpoint
	issueIds := fetchIssueIds(targetUrl, params)
	if len(issueIds) > 0 {
		log.Info().Msgf("Fetched %d issue IDs for sentryIssueDetail", len(issueIds))
	} else {
		log.Warn().Msg("sentryIssueDetail: no issues found during init — targeter will return errors")
	}

	log.Trace().Msgf("SentryIssueDetail generation for:\n%+v", params)
	return &sentryIssueDetailLoadTester{url: targetUrl, params: params, issueIds: issueIds}
}

func fetchIssueIds(baseURL string, params SentryIssueDetailJob) []string {
	common := params.SentryReadAPICommon
	queryParams := url.Values{}
	queryParams.Set("query", "is:unresolved")
	queryParams.Set("limit", fmt.Sprintf("%d", params.FetchLimit))
	queryParams.Set("statsPeriod", "24h")

	path := fmt.Sprintf("/api/0/organizations/%s/issues/", common.OrganizationSlug)
	reqURL := buildQueryURL(baseURL, path, queryParams)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("sentryIssueDetail: failed to build fetch request")
		return nil
	}
	req.Header.Set("Authorization", sentryReadAuthHeader(common))
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("sentryIssueDetail: failed to fetch issue IDs")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("sentryIssueDetail: fetch issue IDs returned status %d", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("sentryIssueDetail: failed to read response body")
		return nil
	}

	var issues []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &issues); err != nil {
		log.Error().Err(err).Msg("sentryIssueDetail: failed to parse issue IDs")
		return nil
	}

	ids := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.ID != "" {
			ids = append(ids, issue.ID)
		}
	}
	return ids
}

func (lt *sentryIssueDetailLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	common := lt.params.SentryReadAPICommon
	issueIds := lt.issueIds
	detailWeight := lt.params.DetailWeight
	latestEventWeight := lt.params.LatestEventWeight
	totalWeight := detailWeight + latestEventWeight

	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if len(issueIds) == 0 {
			return fmt.Errorf("sentryIssueDetail: no issue IDs configured")
		}

		tgt.Method = "GET"
		setSentryReadHeaders(tgt, common)

		issueId := issueIds[rand.Intn(len(issueIds))]

		useLatestEvent := false
		if totalWeight > 0 && latestEventWeight > 0 {
			useLatestEvent = rand.Intn(totalWeight) < latestEventWeight
		}

		var path string
		if useLatestEvent {
			path = fmt.Sprintf("/api/0/organizations/%s/issues/%s/events/latest/", common.OrganizationSlug, issueId)
		} else {
			path = fmt.Sprintf("/api/0/organizations/%s/issues/%s/", common.OrganizationSlug, issueId)
		}

		tgt.URL = buildQueryURL(lt.url, path, url.Values{})
		return nil
	}, 0
}

func (lt *sentryIssueDetailLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {}

func init() {
	RegisterTestType("sentryIssueDetail", newSentryIssueDetailLoadTester, nil)
}
