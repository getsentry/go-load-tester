package tests

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
)

type SentryOrganizationEventsStatsJob struct {
	SentryReadAPICommon `json:",inline"`
	YAxes               [][]string `json:"yAxes,omitempty"`
	Intervals           []string   `json:"intervals,omitempty"`
	Queries             []string   `json:"queries,omitempty"`
	Datasets            []string   `json:"datasets,omitempty"`
}

type sentryOrganizationEventsStatsLoadTester struct {
	url    string
	params SentryOrganizationEventsStatsJob
}

func newSentryOrganizationEventsStatsLoadTester(targetUrl string, rawParams json.RawMessage) LoadTester {
	var params SentryOrganizationEventsStatsJob
	if err := json.Unmarshal(rawParams, &params); err != nil {
		log.Error().Err(err).Msgf("invalid sentryOrganizationEventsStats params received\nraw data\n%s", rawParams)
	}
	log.Trace().Msgf("SentryOrganizationEventsStats generation for:\n%+v", params)
	return &sentryOrganizationEventsStatsLoadTester{url: targetUrl, params: params}
}

func (lt *sentryOrganizationEventsStatsLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	common := lt.params.SentryReadAPICommon
	yAxes := lt.params.YAxes
	intervals := lt.params.Intervals
	queries := lt.params.Queries
	datasets := lt.params.Datasets

	defaultYAxes := []string{"count()"}

	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "GET"
		setSentryReadHeaders(tgt, common)

		params := url.Values{}

		axes := randomStringSliceChoice(yAxes, defaultYAxes)
		for _, axis := range axes {
			params.Add("yAxis", axis)
		}

		params.Set("statsPeriod", randomStringChoice(common.StatsPeriods, "24h"))
		params.Set("interval", randomStringChoice(intervals, "1h"))

		if query := randomStringChoice(queries, ""); query != "" {
			params.Set("query", query)
		}
		if dataset := randomStringChoice(datasets, ""); dataset != "" {
			params.Set("dataset", dataset)
		}

		if projectId := randomProjectId(common); projectId != "" {
			params.Set("project", projectId)
		}

		path := fmt.Sprintf("/api/0/organizations/%s/events-stats/", common.OrganizationSlug)
		tgt.URL = buildQueryURL(lt.url, path, params)
		return nil
	}, 0
}

func (lt *sentryOrganizationEventsStatsLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {}

func init() {
	RegisterTestType("sentryOrganizationEventsStats", newSentryOrganizationEventsStatsLoadTester, nil)
}
