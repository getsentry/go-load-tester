package tests

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
)

type SentryOrganizationGroupIndexJob struct {
	SentryReadAPICommon `json:",inline"`
	SortOptions         []string `json:"sortOptions,omitempty"`
	Queries             []string `json:"queries,omitempty"`
}

type sentryOrganizationGroupIndexLoadTester struct {
	url    string
	params SentryOrganizationGroupIndexJob
}

func newSentryOrganizationGroupIndexLoadTester(targetUrl string, rawParams json.RawMessage) LoadTester {
	var params SentryOrganizationGroupIndexJob
	if err := json.Unmarshal(rawParams, &params); err != nil {
		log.Error().Err(err).Msgf("invalid sentryOrganizationGroupIndex params received\nraw data\n%s", rawParams)
	}
	log.Trace().Msgf("SentryOrganizationGroupIndex generation for:\n%+v", params)
	return &sentryOrganizationGroupIndexLoadTester{url: targetUrl, params: params}
}

func (lt *sentryOrganizationGroupIndexLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	common := lt.params.SentryReadAPICommon
	sortOptions := lt.params.SortOptions
	queries := lt.params.Queries

	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "GET"
		setSentryReadHeaders(tgt, common)

		params := url.Values{}
		params.Set("statsPeriod", randomStringChoice(common.StatsPeriods, "24h"))
		params.Set("sort", randomStringChoice(sortOptions, "date"))
		params.Set("query", randomStringChoice(queries, "is:unresolved"))
		params.Set("limit", fmt.Sprintf("%d", randomIntChoice(common.Limits, 25)))

		if projectId := randomProjectId(common); projectId != "" {
			params.Set("project", projectId)
		}

		path := fmt.Sprintf("/api/0/organizations/%s/issues/", common.OrganizationSlug)
		tgt.URL = buildQueryURL(lt.url, path, params)
		return nil
	}, 0
}

func (lt *sentryOrganizationGroupIndexLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {}

func init() {
	RegisterTestType("sentryOrganizationGroupIndex", newSentryOrganizationGroupIndexLoadTester, nil)
}
