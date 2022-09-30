package tests

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"

	"github.com/getsentry/go-load-tester/utils"
)

// MetricBucketJob is how a metricBucket job is parametrized
//
type MetricBucketJob struct {
	// Total number of metric names that will be generated
	NumMetricNames int

	NumProjects      int
	NumDistributions int
	NumGauges        int
	NumSets          int
	NumCounters      int
	// TODO add the rest configurations
}

type BucketType string

const (
	Distribution BucketType = "d"
	Counter      BucketType = "c"
	Set          BucketType = "s"
	Gauge        BucketType = "g"
)

type MetricBucket struct {
	Name  string `json:"name"`
	Unit  string `json:"unit"`
	Width uint64 `json:"width"`
	// one of
	Type BucketType `json:"type"`
	// Value is []float64 for distribution, float64 for counter, []int32 for set, GaugeValue for gauge
	Value     any               `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

type GaugeValue struct {
	Max   float64 `json:"max"`
	Min   float64 `json:"min"`
	Sum   float64 `json:"sum"`
	Last  float64 `json:"last"`
	Count uint64  `json:"count"`
}

type metricBucketLoadTester struct {
	url                string
	metricBucketParams MetricBucketJob
}

func newMetricsBucketLoadTester(url string, rawTransaction json.RawMessage) LoadTester {
	var metricBucketParams MetricBucketJob
	err := json.Unmarshal(rawTransaction, &metricBucketParams)
	if err != nil {
		log.Error().Err(err).Msgf("invalid metric bucket params received\nraw data\n%s",
			rawTransaction)
	}
	log.Trace().Msgf("MetricBucket generation for:\n%+v", metricBucketParams)

	return &metricBucketLoadTester{
		url:                url,
		metricBucketParams: metricBucketParams,
	}
}

func (mlt *metricBucketLoadTester) GenerateBucket(bucketType BucketType) MetricBucket {
	// TODO fill bucket with values based on the metric parameters

	var timestamp int64 = time.Now().Unix()
	// TODO check
	var width uint64 = 2
	// TODO check
	var unit string = "theunit"
	// TODO check
	var sourceEventType string = "transactions"

	var numMetricNames int = mlt.metricBucketParams.NumMetricNames
	if numMetricNames <= 0 {
		numMetricNames = 1
	}

	var metricName string = fmt.Sprintf("metric%d", rand.Int63n(int64(numMetricNames)))
	var fullMetricName string = fmt.Sprintf("%s:%s/%s@none", bucketType, metricName, sourceEventType)
	tags := map[string]string{
		"name1": "value1",
		"name2": "value2",
	}

	switch bucketType {
	case Distribution:
		return MetricBucket{
			Type: Distribution,
			// TODO....
			Name:      fullMetricName,
			Value:     []float64{1.0, 2.0},
			Unit:      unit,
			Width:     width,
			Timestamp: timestamp,
			Tags:      tags,
		}
	case Set:
		return MetricBucket{
			Type: Set,
			// TODO....
			Name:      fullMetricName,
			Value:     []int32{1, 2, 3},
			Unit:      unit,
			Width:     width,
			Timestamp: timestamp,
			Tags:      tags,
		}
	case Counter:
		return MetricBucket{
			Type: Counter,
			// TODO....
			Name:      fullMetricName,
			Value:     33.0,
			Unit:      unit,
			Width:     width,
			Timestamp: timestamp,
			Tags:      tags,
		}
	case Gauge:
		return MetricBucket{
			Type: Gauge,
			// TODO....
			Name: fullMetricName,
			Value: GaugeValue{
				Min:   1.0,
				Max:   20.0,
				Last:  5.0,
				Sum:   26.0,
				Count: 3,
			},
			Unit:      unit,
			Width:     width,
			Timestamp: timestamp,
			Tags:      tags,
		}
	default:
		panic("Unknown bucket type")
	}
}

func (mlt *metricBucketLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	projectProvider := utils.GetProjectProvider()
	var numProjects = mlt.metricBucketParams.NumProjects
	var numCounters = mlt.metricBucketParams.NumCounters
	var numSets = mlt.metricBucketParams.NumSets
	var numDistributions = mlt.metricBucketParams.NumDistributions
	var numGauges = mlt.metricBucketParams.NumGauges

	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "POST"

		projectId := projectProvider.GetProjectId(numProjects)
		projectInfo := projectProvider.GetProjectInfo(projectId)
		projectKey := projectInfo.ProjectKey

		tgt.URL = fmt.Sprintf("%s/api/%s/envelope/", mlt.url, projectId)
		tgt.Header = make(http.Header)
		tgt.Header.Set("X-Sentry-Auth", utils.GetAuthHeader(projectKey))
		tgt.Header.Set("Content-Type", "application/x-sentry-envelope")

		buckets := make([]MetricBucket, 0, numCounters+numGauges+numDistributions+numSets)

		for i := 0; i < numCounters; i++ {
			bucket := mlt.GenerateBucket(Counter)
			buckets = append(buckets, bucket)
		}
		for i := 0; i < numSets; i++ {
			bucket := mlt.GenerateBucket(Set)
			buckets = append(buckets, bucket)
		}
		for i := 0; i < numDistributions; i++ {
			bucket := mlt.GenerateBucket(Distribution)
			buckets = append(buckets, bucket)
		}
		for i := 0; i < numGauges; i++ {
			bucket := mlt.GenerateBucket(Gauge)
			buckets = append(buckets, bucket)
		}

		body, err := json.Marshal(buckets)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		// TODO figure out the traceId... probably same deal as in transaction generation
		extraEnvelopeHeaders := map[string]string{
			// "trace_id":   transaction.Contexts.Trace.TraceId,
			"public_key": projectKey,
		}

		EventId := EventIdGenerator()()

		// TODO check how we assemble buckets in items
		// probably multiple buckets in one item (but not sure)
		// the code below assumes one item with multiple buckets (need to double check with ingest team)
		buff, err := utils.EnvelopeFromBody(EventId, now, "metric_buckets", extraEnvelopeHeaders, body)
		if err != nil {
			return err
		}

		tgt.Body = buff.Bytes()
		log.Trace().Msgf("Attacking project:%s", projectId)
		return nil
	}, 0
}

func (mlt *metricBucketLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {
	return // nothing to do
}

func init() {
	RegisterTestType("metricBucket", newMetricsBucketLoadTester, nil)
}
