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
// example:
// ```json
// {
//   "numMetricNames": 100,
//   "numProjects": 1000,
//   "numDistributions": 7,
//   "numGauges": 0,
//   "numSets": 3,
//   "numCounters": 8,
//   "minMetricsInDistribution": 4,
//   "maxMetricsInDistribution": 25,
//   "minMetricsInSets": 4,
//   "maxMetricsInSets": 30,
//   "numTagsPerMetric": 5,
//   "numValuesPerTag": 3
// }
// ```
type MetricBucketJob struct {
	// NumMetricNames represents the total number of metric names that will be generated
	NumMetricNames int `json:"numMetricNames"`
	// NumProjects represents the number of projects that will be used for the metric messages
	NumProjects int `json:"numProjects"`
	// NumDistributions created in each messages
	NumDistributions int `json:"numDistributions"`
	// NumberGauges created in each message
	NumGauges int `json:"numGauges"`
	// NumSets created in each messages
	NumSets int `json:"numSets"`
	// NumCounters created in each message
	NumCounters int `json:"numCounters"`
	// MinMetricsInDistribution the minimum number of metrics created in each distribution bucket
	MinMetricsInDistribution int `json:"minMetricsInDistribution"`
	// MaxMetricsInDistribution the maximum number of metrics created in each distribution bucket
	MaxMetricsInDistribution int `json:"maxMetricsInDistribution"`
	// MinMetricsInSets the minimum number of metrics created in each set
	MinMetricsInSets int `json:"minMetricsInSets"`
	// MaxMetricsInSets the maximum number of metrics created in each set
	MaxMetricsInSets int `json:"maxMetricsInSets"`
	// NumTagsPerMetric Number of tags created for each bucket
	// To make things predictable each bucket will contain the all the tags
	// The number of total buckets can be calculated as NumTagsPerMetric^NumValuesPerTag
	NumTagsPerMetric int `json:"numTagsPerMetric"`
	// NumValuesPerTag how many distinct values are generated for each tag
	NumValuesPerTag int `json:"numValuesPerTag"`
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

func randomGaugeValue() GaugeValue {
	min := rand.Float64()*100 + 1
	max := rand.Float64()*100 + min
	last := min + (max-min)*rand.Float64()  // somewhere between max and min
	count := rand.Int63n(50) + 4            // at least a few so max, min and last make some sense
	sum := float64(count) * (max + min) / 2 // something plausible ( count * middle of the interval)
	return GaugeValue{
		Max:   max,
		Min:   min,
		Sum:   sum,
		Last:  last,
		Count: uint64(count),
	}
}

func randomTags(numTags int, numValues int) map[string]string {
	retVal := make(map[string]string, numTags)
	for tagIdx := 1; tagIdx <= numTags; tagIdx++ {
		tagName := fmt.Sprintf("t%d", tagIdx)
		tagValue := fmt.Sprintf("v%d", rand.Intn(numValues)+1)
		retVal[tagName] = tagValue
	}
	return retVal
}

func randomIntArray(minNumElements int, maxNumElements int) []int32 {
	numElements := minNumElements + rand.Intn(maxNumElements-minNumElements)
	retVal := make([]int32, 0, numElements)
	var lastValue int32 = 0
	for idx := 0; idx < numElements; idx++ {
		lastValue += rand.Int31n(5)
		retVal = append(retVal, lastValue)
	}
	return retVal
}

func randomFloat64Array(minNumElements int, maxNumElements int) []float64 {
	numElements := minNumElements + rand.Intn(maxNumElements-minNumElements)
	retVal := make([]float64, 0, numElements)
	lastValue := 0.0
	for idx := 0; idx < numElements; idx++ {
		lastValue += rand.Float64() * 5.0
		retVal = append(retVal, lastValue)
	}
	return retVal
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

	var timestamp int64 = time.Now().Unix()
	var width uint64 = 2
	var unit string = "theunit"
	var sourceEventType string = "transactions"
	var params MetricBucketJob = mlt.metricBucketParams

	var numMetricNames int = params.NumMetricNames
	if numMetricNames <= 0 {
		numMetricNames = 1
	}

	var metricName string = fmt.Sprintf("metric%d", rand.Int63n(int64(numMetricNames)))
	var fullMetricName string = fmt.Sprintf("%s:%s/%s@none", bucketType, sourceEventType, metricName)
	tags := randomTags(params.NumTagsPerMetric, params.NumValuesPerTag)

	switch bucketType {
	case Distribution:
		return MetricBucket{
			Type:      Distribution,
			Name:      fullMetricName,
			Value:     randomFloat64Array(params.MinMetricsInDistribution, params.MaxMetricsInDistribution),
			Unit:      unit,
			Width:     width,
			Timestamp: timestamp,
			Tags:      tags,
		}
	case Set:
		return MetricBucket{
			Type:      Set,
			Name:      fullMetricName,
			Value:     randomIntArray(params.MaxMetricsInSets, params.MinMetricsInSets),
			Unit:      unit,
			Width:     width,
			Timestamp: timestamp,
			Tags:      tags,
		}
	case Counter:
		return MetricBucket{
			Type:      Counter,
			Name:      fullMetricName,
			Value:     33.0,
			Unit:      unit,
			Width:     width,
			Timestamp: timestamp,
			Tags:      tags,
		}
	case Gauge:
		return MetricBucket{
			Type:      Gauge,
			Name:      fullMetricName,
			Value:     randomGaugeValue(),
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
	var traceGenerator = EventIdGenerator()

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
		extraEnvelopeHeaders := map[string]string{
			"trace_id":   traceGenerator(),
			"public_key": projectKey,
		}

		EventId := EventIdGenerator()()

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
