package tests

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
)

// Contains  functionality for generating Session load tests

type ClickhouseQueryJob struct {
	// A random int that changes the SELECT
	multiplier int64
}

type clickhouseQueryLoadTester struct {
	url         string
	queryParams ClickhouseQueryJob
}

func newClickhouseQueryLoadTester(url string, rawClickhouseQueryParams json.RawMessage) LoadTester {
	var jsonClickhouseQueryParams ClickhouseQueryJobRaw
	err := json.Unmarshal(rawClickhouseQueryParams, &jsonClickhouseQueryParams)
	if err != nil {
		log.Error().Err(err).Msgf("invalid Clickhouse Query params received\nraw data\n%s",
			rawClickhouseQueryParams)
	}

	var clickhouseQueryParams ClickhouseQueryJob
	jsonClickhouseQueryParams.into(&clickhouseQueryParams)

	log.Trace().Msgf("Clickhouse Query generation for:\n%+v", clickhouseQueryParams)

	return &clickhouseQueryLoadTester{
		url:         url,
		queryParams: clickhouseQueryParams,
	}
}

func (slt *clickhouseQueryLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "GET"
		args := url.QueryEscape(fmt.Sprintf("SELECT %d;", slt.queryParams.multiplier))
		tgt.URL = fmt.Sprintf("%s/?query=%s", slt.url, args)
		return nil
	}, 0
}

func (slt *clickhouseQueryLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {
	return // nothing to do
}

type ClickhouseQueryJobRaw struct {
	Multiplier int64 `json:"multiplier"`
}

func (raw ClickhouseQueryJobRaw) into(result *ClickhouseQueryJob) error {
	result.multiplier = raw.Multiplier
	return nil
}

func init() {
	RegisterTestType("clickhouseQuery", newClickhouseQueryLoadTester, nil)
}
