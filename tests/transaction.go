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

// TransactionJobCommon represents the common parameters for transactions jobs (both V1 and V2)
// NOTE: No YAML support (yaml unlike json does not flatten embedded structures)
type TransactionJobCommon struct {
	// TransactionDurationMax the maximum duration for a transactionJob
	TransactionDurationMax utils.StringDuration `json:"transactionDurationMax,omitempty" yaml:"transactionDurationMax,omitempty"`
	// TransactionDurationMin the minimum duration for a transactionJob
	TransactionDurationMin utils.StringDuration `json:"transactionDurationMin,omitempty" yaml:"transactionDurationMin,omitempty"`
	// MinSpans specifies the minimum number of spans generated in a transactionJob
	MinSpans uint64 `json:"minSpans,omitempty" yaml:"minSpans,omitempty"`
	// MaxSpans specifies the maximum number of spans generated in a transactionJob
	MaxSpans uint64 `json:"maxSpans,omitempty" yaml:"maxSpans,omitempty"`
	// NumReleases specifies the maximum number of unique releases generated in a test
	NumReleases uint64 `json:"numReleases,omitempty" yaml:"numReleases,omitempty"`
	// NumUsers specifies the maximum number of unique users generated in a test
	NumUsers uint64 `json:"numUsers,omitempty" yaml:"numUsers,omitempty"`
	// MinBreadcrumbs specifies the minimum number of breadcrumbs that will be generated in a test
	MinBreadcrumbs uint64 `json:"minBreadcrumbs,omitempty" yaml:"minBreadcrumbs,omitempty"`
	// MaxBreadcrumbs specifies the maximum number of breadcrumbs that will be generated in a test
	MaxBreadcrumbs uint64 `json:"maxBreadcrumbs,omitempty" yaml:"maxBreadcrumbs,omitempty"`
	// BreadcrumbCategories the categories used for breadcrumbs (if not specified defaults will be used *)
	BreadcrumbCategories []string `json:"breadcrumbCategories,omitempty" yaml:"breadcrumbCategories,omitempty"`
	// BreadcrumbLevels specifies levels used for breadcrumbs (if not specified defaults will be used *)
	BreadcrumbLevels []string `json:"breadcrumbLevels,omitempty" yaml:"breadcrumbLevels,omitempty"`
	// BreadcrumbsTypes specifies the types used for breadcrumbs (if not specified defaults will be used *)
	BreadcrumbsTypes []string `json:"breadcrumbsTypes,omitempty" yaml:"breadcrumbsTypes,omitempty"`
	// BreadcrumbMessages specifies messages set in breadcrumbs (if not specified defaults will be used *)
	BreadcrumbMessages []string `json:"breadcrumbMessages,omitempty" yaml:"breadcrumbMessages,omitempty"`
	// Measurements specifies measurements to be used (if not specified NO measurements will be generated)
	Measurements []string `json:"measurements,omitempty" yaml:"measurements,omitempty"`
	// Operations specifies the operations to be used (if not specified NO operations will be generated)
	Operations []string `json:"operations,omitempty" yaml:"operations,omitempty"`
}

// TransactionJob is how a transactionJob load test is parameterized
// example:
// ```json
// {
//  "numProjects": 10000,
//  "transactionDurationMax":"10m" ,
//  "transactionDurationMin": "1m" ,
//  "transactionTimestampSpread": "5h" ,
//  "minSpans": 5,
//  "maxSpans": 40,
//  "numReleases": 1000 ,
//  "numUsers": 2000,
//  "minBreadcrumbs": 5,
//  "maxBreadcrumbs": 25,
//  "breadcrumbCategories": ["auth", "web-request", "query"],
//  "breadcrumbLevels": ["fatal", "error", "warning", "info", "debug"],
//  "breadcrumbsTypes": ["default", "http", "error"] ,
//  "breadcrumbMessages": [ "Authenticating the user_name", "IOError: [Errno 2] No such file"],
//  "measurements": ["fp","fcp","lcp","fid","cls","ttfb"],
//  "operations": ["browser","http","db","resource.script"]
// }
//
//  NOTE YAML serializer creates a sub-object for embedded structures (i.e. all fields in TransactionJobCommon
//  will appear as
// ```
type TransactionJob struct {
	// NumProjects to use in the requests
	NumProjects int `json:"numProjects" yaml:"numProjects"`
	// TransactionTimestampSpread the spread (from Now) of the timestamp, generated transactions will have timestamps between
	// `Now` and `Now-TransactionTimestampSpread`
	TransactionTimestampSpread utils.StringDuration `json:"transactionTimestampSpread,omitempty" yaml:"transactionTimestampSpread,omitempty"`
	// TransactionJobCommon Common transaction job parameters
	TransactionJobCommon `yaml:"transactionJobCommon"`
}

// TimestampHistogramBucket represents a bucket in a timestamp histogram
// An array of buckets fully defines a timestamp histogram as in the example below.
// e.g. consider the following histogram (in json format):
// ```json
// [
//    { "ratio": 5, "maxDelay": "1s"},
//    { "ratio": 3, "maxDelay": "10s"},
//	  { "ratio": 2, "maxDelay": "20s"}
// ]
//  In the example below we have the cumulative ratio 5 + 3 + 2 = 10
//  The request will be generated so that, on average, for every 10 requests
//  5 will be in the first bucket, 3 in the second and the rest of 2 in the third,
//  or in other words 50% of the requests will have a timestamp delay of between 0s and 1s,
//  30% will have a timestamp delay of between 1s and 10s and 20% between 10s and 20s
//  **Note:** the ratio can be any positive numbers (including 0 if you want to skip an interval),
//  they do not have to add up to 10 (as in the example) or any other number, the
//  same ratio would be achieved by using 0.5,0.3,0.2 or 50,30,20.
// ```
type TimestampHistogramBucket struct {
	// Ratio is the relative frequency of requests in this bucket, relative to all other buckets
	Ratio float64 `json:"ratio" yaml:"ratio"`
	// MaxDelay The upper bound of the bucket, the lower bound is the previous bucket upper bound or 0 for the first bucket
	MaxDelay utils.StringDuration `json:"maxDelay" yaml:"maxDelay"`
}

// ProjectProfile represents a group of projects with the same relative frequency and the same timestamp histogram
// A request will get an array of ProjectProfile, the example below further explains how this works:
// Consider the following example in JSON format (the timestamp histogram was removed since it is not relevant in the
// current explanation).
// ```json
// [
//	 { "numProjects": 2, "relativeFreqRatio": 4},
//	 { "numProjects": 3, "relativeFreqRatio": 2},
//	 { "numProjects": 4, "relativeFreqRatio": 1},
// ```
// In the example above we have 3 groups, the total number of projects generated is 10+1+5 = 16
// Events for a particular project in the first bucket will occur twice as often as events for a
// particular event in the second bucket and four times as frequent as events for a particular
// project in the second bucket.
// In other words considering that projects 1,2 belong to bucket 1, projects 3,4,5 to bucket 2 and
// projects 6,7,8,9 to bucket 3 here's a perfect distribution of events for the profile above
// 1 1 1 1  2 2 2 2  3 3  4 4  5  5  6  7  8  9
//
type ProjectProfile struct {
	// NumProjects number of projects that use this profile
	NumProjects int `json:"numProjects" yaml:"numProjects"`
	// Relative frequency of project from this profile in relation with projects from other profiles
	RelativeFreqRatio float64 `json:"relativeFreqRatio" yaml:"relativeFreqRatio"`
	// The timestamp histogram for projects in this profile
	TimestampHistogram []TimestampHistogramBucket `json:"timestampProfiles" yaml:"timestampProfiles"`
}

func (pp ProjectProfile) GetNumProjects() int {
	return pp.NumProjects
}

func (pp ProjectProfile) GetRelativeFreqRatio() float64 {
	return pp.RelativeFreqRatio
}

// TransactionJobV2 is how a transactionJobV2 load test is parameterized
// example:
// ```json
// {
//  "projectDistribution: [
//    {
//      "numProjects": 100,
//      "relativeFreqRatio" : 1.0,
//      "timestampHistogram": [
//        { "ratio": 5, "maxDelay": "1s"},
//        { "ratio": 3, "maxDelay": "10s"},
//	      { "ratio": 2, "maxDelay": "20s"}
//      ]
//    },
//    {
//      "numProjects": 200,
//      "relativeFreqRatio" : 4.0,
//      "timestampHistogram": [
//        { "ratio": 20, "maxDelay": "1s"},
//        { "ratio": 1, "maxDelay": "5s"}
//      ]
//    }
//  ],
//  "transactionDurationMax":"10m" ,
//  "transactionDurationMin": "1m" ,
//  "minSpans": 5,
//  "maxSpans": 40,
//  "numReleases": 1000 ,
//  "numUsers": 2000,
//  "minBreadcrumbs": 5,
//  "maxBreadcrumbs": 25,
//  "breadcrumbCategories": ["auth", "web-request", "query"],
//  "breadcrumbLevels": ["fatal", "error", "warning", "info", "debug"],
//  "breadcrumbsTypes": ["default", "http", "error"] ,
//  "breadcrumbMessages": [ "Authenticating the user_name", "IOError: [Errno 2] No such file"],
//  "measurements": ["fp","fcp","lcp","fid","cls","ttfb"],
//  "operations": ["browser","http","db","resource.script"]
// }
// ```
//
//  NOTE YAML serializer creates a sub-object for embedded structures (i.e. all fields in TransactionJobCommon
//  will appear as
// ```

type TransactionJobV2 struct {
	// The project profiles
	ProjectDistribution []ProjectProfile
	// TransactionJobCommon Common transaction job parameters
	TransactionJobCommon `yaml:"transactionJobCommon"`
}

// transactionLoadTester is used to drive a transaction load test
type transactionLoadTester struct {
	url                   string
	transactionParams     TransactionJobCommon
	transactionGenerator  func(duration time.Duration) Transaction
	version               int
	numProjectsV1         int
	timestampSpreadV1     time.Duration
	projectDistributionV2 []ProjectProfile
}

// newTransactionLoadTester creates a LoadTester for the specified transaction parameters and url
func newTransactionLoadTester(url string, rawTransaction json.RawMessage) LoadTester {
	var transactionParams TransactionJob
	err := json.Unmarshal(rawTransaction, &transactionParams)
	if transactionParams.NumProjects == 0 {
		// backward compatibility (if nothing provided fall back on one project)
		transactionParams.NumProjects = 1
	}

	if err != nil {
		log.Error().Err(err).Msgf("invalid transaction params received\nraw data\n%s",
			rawTransaction)
	}
	log.Trace().Msgf("Transaction generation for:\n%+v", transactionParams)

	transactionGenerator := TransactionGenerator(transactionParams.TransactionJobCommon)

	return &transactionLoadTester{
		transactionGenerator: transactionGenerator,
		url:                  url,
		transactionParams:    transactionParams.TransactionJobCommon,
		version:              1,
		numProjectsV1:        transactionParams.NumProjects,
		timestampSpreadV1:    time.Duration(transactionParams.TransactionTimestampSpread),
	}
}

// newTransactionLoadTester creates a LoadTester for the specified transaction parameters and url
func newTransactionLoadTesterV2(url string, rawTransaction json.RawMessage) LoadTester {
	var transactionParams TransactionJobV2
	err := json.Unmarshal(rawTransaction, &transactionParams)

	if err != nil {
		log.Error().Err(err).Msgf("invalid transaction params received\nraw data\n%s",
			rawTransaction)
	}
	log.Trace().Msgf("Transaction generation for:\n%+v", transactionParams)

	transactionGenerator := TransactionGenerator(transactionParams.TransactionJobCommon)

	return &transactionLoadTester{
		transactionGenerator:  transactionGenerator,
		url:                   url,
		transactionParams:     transactionParams.TransactionJobCommon,
		version:               2,
		projectDistributionV2: transactionParams.ProjectDistribution,
	}
}

// timeSpreadGenerator is used to generate time spreads in accordance to the ProjectProfiles from
// the specified request
// The function returned accepts a project profile index (generated by some logic outside the function)
// and uses the histogram for the selected project profile to generate a delay
func timeSpreadGenerator(projectProfiles []ProjectProfile) func(profileIdx int) time.Duration {

	// rearrange histograms in a better way for generation, accumulate values from the left, i.e.
	// calculate integral.
	type cumulatedRatio struct {
		upTo     float64
		maxDelay time.Duration
	}
	profiles := make([][]cumulatedRatio, 0, len(projectProfiles))
	for profileIdx := 0; profileIdx < len(profiles); profileIdx++ {
		currentHistogram := projectProfiles[profileIdx].TimestampHistogram
		val := make([]cumulatedRatio, 0, len(currentHistogram))
		var acc float64
		for histIdx := 0; histIdx < len(currentHistogram); histIdx++ {
			acc += currentHistogram[histIdx].Ratio
			ratio := cumulatedRatio{
				upTo:     acc,
				maxDelay: time.Duration(currentHistogram[histIdx].MaxDelay),
			}
			val = append(val, ratio)
		}
		profiles = append(profiles, val)
	}

	return func(profileIdx int) time.Duration {
		histogram := profiles[profileIdx]
		maxVal := histogram[len(histogram)-1].upTo
		val := rand.Float64() * maxVal
		// first bucket starts at delay=0
		var lowerBound int64 = 0
		for idx := 0; idx < len(histogram); idx++ {
			if histogram[idx].upTo <= val {
				upperBound := int64(histogram[idx].maxDelay)
				// get a delay within our histogram bucket
				delay := lowerBound + rand.Int63n(upperBound-lowerBound)
				return time.Duration(delay)
			}
			// update lower bound for the next bucket
			lowerBound = int64(histogram[idx].maxDelay)
		}
		// should never get here
		panic("Failed to calculate delay in histogram")
	}
}

func (tlt *transactionLoadTester) GetTargeter() (vegeta.Targeter, uint64) {
	projectProvider := utils.GetProjectProvider()
	var getProjectIdAndTimestampDelay func() (string, time.Duration, error)

	if tlt.version == 1 {
		getProjectIdAndTimestampDelay = func() (string, time.Duration, error) {
			return projectProvider.GetProjectId(tlt.numProjectsV1), tlt.timestampSpreadV1, nil
		}
	} else if tlt.version == 2 {
		projectProfiles := tlt.projectDistributionV2
		projectDistribution := make([]utils.ProjectFreqProfile, 0, len(projectProfiles))
		for idx := 0; idx < len(projectProfiles); idx++ {
			projectDistribution = append(projectDistribution, projectProfiles[idx])
		}
		generator := timeSpreadGenerator(tlt.projectDistributionV2)
		getProjectIdAndTimestampDelay = func() (string, time.Duration, error) {
			projectId, profileIdx, err := projectProvider.GetProjectIdV2(projectDistribution)
			timestamp := generator(profileIdx)
			if err != nil {
				log.Error().Err(err).Msg("Could not get project id from project provider")
				return "", timestamp, err
			}
			return projectId, time.Second, err
		}
	}

	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "POST"

		projectId, timeSpread, err := getProjectIdAndTimestampDelay()

		if err != nil {
			return err
		}

		projectInfo := projectProvider.GetProjectInfo(projectId)
		projectKey := projectInfo.ProjectKey

		tgt.URL = fmt.Sprintf("%s/api/%s/envelope/", tlt.url, projectId)
		tgt.Header = make(http.Header)
		tgt.Header.Set("X-Sentry-Auth", utils.GetAuthHeader(projectKey))
		tgt.Header.Set("Content-Type", "application/x-sentry-envelope")

		transaction := tlt.transactionGenerator(timeSpread)

		body, err := json.Marshal(transaction)
		if err != nil {
			return err
		}

		// var buff *bytes.Buffer
		now := time.Now().UTC()
		extraEnvelopeHeaders := map[string]string{
			"trace_id":   transaction.Contexts.Trace.TraceId,
			"public_key": projectKey,
		}
		buff, err := utils.EnvelopeFromBody(transaction.EventId, now, "transaction", extraEnvelopeHeaders, body)
		if err != nil {
			return err
		}

		tgt.Body = buff.Bytes()
		log.Trace().Msgf("Attacking project:%s", projectId)
		return nil
	}, 0
}

func (tlt *transactionLoadTester) ProcessResult(_ *vegeta.Result, _ uint64) {
	return // nothing to do
}

// Transaction defines the JSON format of a Sentry transactionJob,
// NOTE: this is just part of a Sentry Event, if we need to emit
// other Events convert this structure into an Event struct and
// add the other fields to it .
type Transaction struct {
	Timestamp      string             `json:"timestamp,omitempty"`       // RFC 3339
	StartTimestamp string             `json:"start_timestamp,omitempty"` // RFC 3339
	EventId        string             `json:"event_id"`
	Release        string             `json:"release,omitempty"`
	Transaction    string             `json:"transactionJob,omitempty"`
	Logger         string             `json:"logger,omitempty"`
	Environment    string             `json:"environment,omitempty"`
	User           User               `json:"user,omitempty"`
	Contexts       Contexts           `json:"contexts,omitempty"`
	Breadcrumbs    []Breadcrumb       `json:"breadcrumbs,omitempty"`
	Measurements   map[string]float64 `json:"measurements,omitempty"`
	Spans          []Span             `json:"spans,omitempty"`
}

func TransactionGenerator(job TransactionJobCommon) func(time.Duration) Transaction {
	idGen := EventIdGenerator()
	relGen := ReleaseGenerator(job.NumReleases)
	transGen := func() string {
		if Flip() {
			return ""
		} else {
			return fmt.Sprintf("mytransaction%d", rand.Intn(100))
		}
	}
	userGen := UserGenerator(job.NumUsers)
	osGen := OsContextGenerator()
	deviceGen := DeviceContextGenerator()
	appGen := AppContextGenerator()
	traceGen := TraceContextGenerator(job.Operations)
	breadcrumbsGen := BreadcrumbsGenerator(job.MinBreadcrumbs, job.MaxBreadcrumbs, job.BreadcrumbCategories,
		job.BreadcrumbLevels, job.BreadcrumbsTypes, job.BreadcrumbMessages)
	measurementsGen := MeasurementsGenerator(job.Measurements)
	spansGen := SpansGenerator(job.MinSpans, job.MaxSpans, job.Operations)

	transactionDurationMin := time.Duration(job.TransactionDurationMin)
	transactionDurationMax := time.Duration(job.TransactionDurationMax)

	transactionRange := transactionDurationMax - transactionDurationMin

	return func(transactionDelta time.Duration) Transaction {
		trace := traceGen()
		transactionId := trace.SpanId
		traceId := trace.TraceId

		now := time.Now()
		transactionDuration := time.Duration(float64(transactionRange) * rand.Float64())
		timestamp := now.Add(-transactionDelta)
		startTimestamp := timestamp.Add(-transactionDuration)

		retVal := Transaction{
			Timestamp:      toUtcString(timestamp),
			StartTimestamp: toUtcString(startTimestamp),
			EventId:        idGen(),
			Release:        relGen(),
			Transaction:    transGen(),
			Logger:         utils.SimpleRandomChoice([]string{"foo.bar.baz", "bam.baz.bad", ""}),
			Environment:    utils.SimpleRandomChoice([]string{"production", "development", "staging"}),
			User:           userGen(),
			Contexts: Contexts{
				Os:     osGen(),
				Device: deviceGen(),
				App:    appGen(),
				Trace:  trace,
			},
			Breadcrumbs:  breadcrumbsGen(),
			Measurements: measurementsGen(),
			Spans:        spansGen(transactionId, traceId, startTimestamp, timestamp),
		}

		return retVal
	}
}

func init() {
	RegisterTestType("transaction", newTransactionLoadTester, nil)
	RegisterTestType("transactionV2", newTransactionLoadTesterV2, nil)
}
