package tests

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
)

type SentryOrganizationEventsJob struct {
	SentryReadAPICommon `json:",inline"`
	FieldSets           [][]string `json:"fieldSets,omitempty"`
	Datasets            []string   `json:"datasets,omitempty"`
	Queries             []string   `json:"queries,omitempty"`
	SortBy              []string   `json:"sortBy,omitempty"`
}

type sentryOrganizationEventsLoadTester struct {
	url    string
	params SentryOrganizationEventsJob
}

func newSentryOrganizationEventsLoadTester(targetUrl string, rawParams json.RawMessage) LoadTester {
	var params SentryOrganizationEventsJob
	if err := json.Unmarshal(rawParams, &params); err != nil {
		log.Error().Err(err).Msgf("invalid sentryOrganizationEvents params received\nraw data\n%s", rawParams)
	}
	log.Trace().Msgf("SentryOrganizationEvents generation for:\n%+v", params)
	return &sentryOrganizationEventsLoadTester{url: targetUrl, params: params}
}

func (lt *sentryOrganizationEventsLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	common := lt.params.SentryReadAPICommon
	fieldSets := lt.params.FieldSets
	datasets := lt.params.Datasets
	queries := lt.params.Queries
	sortBy := lt.params.SortBy

	defaultFields := []string{"title", "count()"}

	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "GET"
		setSentryReadHeaders(tgt, common)

		params := url.Values{}

		fields := randomStringSliceChoice(fieldSets, defaultFields)
		for _, field := range fields {
			params.Add("field", field)
		}

		params.Set("statsPeriod", randomStringChoice(common.StatsPeriods, "24h"))
		params.Set("per_page", fmt.Sprintf("%d", randomIntChoice(common.Limits, 25)))

		if dataset := randomStringChoice(datasets, ""); dataset != "" {
			params.Set("dataset", dataset)
		}
		if query := randomStringChoice(queries, ""); query != "" {
			params.Set("query", query)
		}
		if sort := randomStringChoice(sortBy, ""); sort != "" {
			params.Set("sort", sort)
		}

		if projectId := randomProjectId(common); projectId != "" {
			params.Set("project", projectId)
		}

		path := fmt.Sprintf("/api/0/organizations/%s/events/", common.OrganizationSlug)
		tgt.URL = buildQueryURL(lt.url, path, params)
		return nil
	}, 0
}

func (lt *sentryOrganizationEventsLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {}

func init() {
	RegisterTestType("sentryOrganizationEvents", newSentryOrganizationEventsLoadTester, nil)
}
