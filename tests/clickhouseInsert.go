package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/getsentry/go-load-tester/tests/dataproviders"
	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
)

// Contains  functionality for generating Session load tests

type ClickhouseInsertLoadTester struct {
	url         string
	queryParams dataproviders.ClickhouseInsertJob

	batchBuilder dataproviders.BatchBuilder
}

func newClickhouseInsertLoadTester(url string, rawClickhouseQueryParams json.RawMessage) LoadTester {
	var jsonClickhouseQueryParams dataproviders.ClickhouseInsertJobRaw
	err := json.Unmarshal(rawClickhouseQueryParams, &jsonClickhouseQueryParams)
	if err != nil {
		log.Error().Err(err).Msgf("invalid Clickhouse Insert params received\nraw data\n%s",
			rawClickhouseQueryParams)
	}

	var clickhouseQueryParams dataproviders.ClickhouseInsertJob
	err = jsonClickhouseQueryParams.Into(&clickhouseQueryParams)
	if err != nil {
		log.Error().Err(err).Msgf("invalid Clickhouse Insert params received\nraw data")
	}

	log.Trace().Msgf("Clickhouse Insert generation for:\n%+v", clickhouseQueryParams)

	return &ClickhouseInsertLoadTester{
		url:         url,
		queryParams: clickhouseQueryParams,
		batchBuilder: *dataproviders.NewBatchBuilder(
			clickhouseQueryParams.Schema,
			uint64(clickhouseQueryParams.BatchSize),
		),
	}
}

func (slt *ClickhouseInsertLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "POST"
		log.Trace().Msgf("%v Preparing batch", time.Now().Format("2006-01-02T15:04:05"))
		batch := slt.batchBuilder.BuildBatch()
		var buffer bytes.Buffer
		log.Trace().Msgf("%v Batch Full", time.Now().Format("2006-01-02T15:04:05"))
		for _, row := range batch {
			json_row, err := json.Marshal(row)
			if err != nil {
				log.Error().Err(err).Msgf("Failure in marshalling data")
			}
			buffer.Write(json_row)
			buffer.WriteByte('\n')
		}
		log.Trace().Msgf("%v Batch Serialized", time.Now().Format("2006-01-02T15:04:05"))
		args := url.QueryEscape(fmt.Sprintf("INSERT INTO %s FORMAT JSONEachRow", slt.queryParams.TableName))
		tgt.URL = fmt.Sprintf("%s/?query=%s", slt.url, args)
		tgt.Header = make(http.Header)
		tgt.Header.Set("Content-Type", "application/json")
		tgt.Header.Set("Accept-Encoding", "gzip,deflate")
		tgt.Body = buffer.Bytes()
		return nil
	}, 0
}

func (slt *ClickhouseInsertLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {
	return // nothing to do
}

func clickhouseInsertLoadSplitter(masterParams TestParams, numWorkers int) ([]TestParams, error) {
	if numWorkers <= 0 {
		return nil, fmt.Errorf("invalid number of workers %d need at least 1", numWorkers)
	}

	newParams := masterParams
	newParams.Per = time.Duration(numWorkers) * masterParams.Per
	var jsonClickhouseQueryParams dataproviders.ClickhouseInsertJobRaw
	err := json.Unmarshal(masterParams.Params, &jsonClickhouseQueryParams)
	if err != nil {
		log.Error().Err(err).Msgf("invalid Clickhouse Insert params received\nraw data\n%s",
			masterParams.Params)
	}

	retVal := make([]TestParams, 0, numWorkers)
	for idx := 0; idx < numWorkers; idx++ {
		jsonClickhouseQueryParams.Partitions = numWorkers
		jsonClickhouseQueryParams.PartitionId = idx
		newParams.Params, _ = json.Marshal(jsonClickhouseQueryParams)
		retVal = append(retVal, newParams)
	}
	return retVal, nil
}

func init() {
	RegisterTestType("clickhouseInsert", newClickhouseInsertLoadTester, clickhouseInsertLoadSplitter)
}
