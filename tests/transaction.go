package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/go-load-tester/utils"
	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
	"gopkg.in/yaml.v2"
	"math/rand"
	"net/http"
	"time"
)

// TransactionJob is how a transactionJob load test is parameterized
type TransactionJob struct {
	//TransactionDurationMax the maximum duration for a transactionJob
	TransactionDurationMax time.Duration `json:"transactionDurationMax,omitempty" yaml:"transactionDurationMax,omitempty"`
	//TransactionDurationMin the minimum duration for a transactionJob
	TransactionDurationMin time.Duration `json:"transactionDurationMin,omitempty" yaml:"transactionDurationMin,omitempty"`
	//TransactionTimestampSpread the spread (from Now) of the timestamp, generated transactions will have timestamps between
	//`Now` and `Now-TransactionTimestampSpread`
	TransactionTimestampSpread time.Duration `json:"transactionTimestampSpread,omitempty" yaml:"transactionTimestampSpread,omitempty"`
	//MinSpans specifies the minimum number of spans generated in a transactionJob
	MinSpans uint64 `json:"minSpans,omitempty" yaml:"minSpans,omitempty"`
	//MaxSpans specifies the maximum number of spans generated in a transactionJob
	MaxSpans uint64 `json:"maxSpans,omitempty" yaml:"maxSpans,omitempty"`
	//NumReleases specifies the maximum number of unique releases generated in a test
	NumReleases uint64 `json:"numReleases,omitempty" yaml:"numReleases,omitempty"`
	//NumUsers specifies the maximum number of unique users generated in a test
	NumUsers uint64 `json:"numUsers,omitempty" yaml:"numUsers,omitempty"`
	//MinBreadcrumbs specifies the minimum number of breadcrumbs that will be generated in a test
	MinBreadcrumbs uint64 `json:"minBreadcrumbs,omitempty" yaml:"minBreadcrumbs,omitempty"`
	//MaxBreadcrumbs specifies the maximum number of breadcrumbs that will be generated in a test
	MaxBreadcrumbs uint64 `json:"maxBreadcrumbs,omitempty" yaml:"maxBreadcrumbs,omitempty"`
	// BreadcrumbCategories the categories used for breadcrumbs (if not specified defaults will be used *)
	BreadcrumbCategories []string `json:"breadcrumbCategories,omitempty" yaml:"breadcrumbCategories,omitempty"`
	//BreadcrumbLevels specifies levels used for breadcrumbs (if not specified defaults will be used *)
	BreadcrumbLevels []string `json:"breadcrumbLevels,omitempty" yaml:"breadcrumbLevels,omitempty"`
	//BreadcrumbsTypes specifies the types used for breadcrumbs (if not specified defaults will be used *)
	BreadcrumbsTypes []string `json:"breadcrumbsTypes,omitempty" yaml:"breadcrumbsTypes,omitempty"`
	//BreadcrumbMessages specifies messages set in breadcrumbs (if not specified defaults will be used *)
	BreadcrumbMessages []string `json:"breadcrumbMessages,omitempty" yaml:"breadcrumbMessages,omitempty"`
	//Measurements specifies measurements to be used (if not specified NO measurements will be generated)
	Measurements []string `json:"measurements,omitempty" yaml:"measurements,omitempty"`
	//Operations specifies the operations to be used (if not specified NO operations will be generated)
	Operations []string `json:"operations,omitempty" yaml:"operations,omitempty"`
}

type transactionLoadTester struct {
	url                  string
	transactionGenerator func() Transaction
}

// newTransactionLoadTester creates a LoadTester for the specified transaction parameters and url
func newTransactionLoadTester(url string, rawTransaction json.RawMessage) LoadTester {
	var transactionParams TransactionJob
	err := json.Unmarshal(rawTransaction, &transactionParams)

	if err != nil {
		log.Error().Err(err).Msgf("invalid transaction params received\nraw data\n%s",
			rawTransaction)
	}

	transactionGenerator := TransactionGenerator(transactionParams)

	return &transactionLoadTester{
		transactionGenerator: transactionGenerator,
		url:                  url,
	}
}

func (tlt *transactionLoadTester) GetTargeter() vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "POST"

		//TODO add more sophisticated projectId/projectKey generation
		projectId := "1"
		projectKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"

		tgt.URL = fmt.Sprintf("%s/api/%s/envelope/", tlt.url, projectId)
		tgt.Header = make(http.Header)
		tgt.Header.Set("X-Sentry-Auth", utils.GetAuthHeader(projectKey))
		tgt.Header.Set("Content-Type", "application/x-sentry-envelope")

		transaction := tlt.transactionGenerator()

		body, err := json.Marshal(transaction)
		if err != nil {
			return err
		}

		//var buff *bytes.Buffer
		now := time.Now().UTC()
		buff, err := utils.EnvelopeFromBody(transaction.EventId, now, "transaction", body)
		if err != nil {
			return err
		}

		tgt.Body = buff.Bytes()
		log.Trace().Msg("Attacking")
		return nil
	}
}

func (tlt *transactionLoadTester) ProcessResult(_ *vegeta.Result) {
	return // nothing to do
}

// Transaction defines the JSON format of a Sentry transactionJob,
// NOTE: this is just part of a Sentry Event, if we need to emit
// other Events convert this structure into an Event struct and
// add the other fields to it .
type Transaction struct {
	Timestamp      string             `json:"timestamp,omitempty"`       //RFC 3339
	StartTimestamp string             `json:"start_timestamp,omitempty"` //RFC 3339
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

func TransactionGenerator(job TransactionJob) func() Transaction {
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

	transactionDurationMin := job.TransactionDurationMin
	transactionDurationMax := job.TransactionDurationMax
	transactionTimestampSpread := job.TransactionTimestampSpread
	transactionRange := transactionDurationMax - transactionDurationMin

	return func() Transaction {
		trace := traceGen()
		transactionId := trace.SpanId
		traceId := trace.TraceId

		now := time.Now()
		transactionDuration := time.Duration(float64(transactionRange) * rand.Float64())
		transactionDelta := time.Duration(float64(transactionTimestampSpread) * rand.Float64())
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

type transactionJobRaw struct {
	TransactionDurationMax     string   `json:"transactionDurationMax,omitempty" yaml:"transactionDurationMax,omitempty"`
	TransactionDurationMin     string   `json:"transactionDurationMin,omitempty" yaml:"transactionDurationMin,omitempty"`
	TransactionTimestampSpread string   `json:"transactionTimestampSpread,omitempty" yaml:"transactionTimestampSpread,omitempty"`
	MinSpans                   uint64   `json:"minSpans,omitempty" yaml:"minSpans,omitempty"`
	MaxSpans                   uint64   `json:"maxSpans,omitempty" yaml:"maxSpans,omitempty"`
	NumReleases                uint64   `json:"numReleases,omitempty" yaml:"numReleases,omitempty"`
	NumUsers                   uint64   `json:"numUsers,omitempty" yaml:"numUsers,omitempty"`
	MinBreadcrumbs             uint64   `json:"minBreadcrumbs,omitempty" yaml:"minBreadcrumbs,omitempty"`
	MaxBreadcrumbs             uint64   `json:"maxBreadcrumbs,omitempty" yaml:"maxBreadcrumbs,omitempty"`
	BreadcrumbCategories       []string `json:"breadcrumbCategories,omitempty" yaml:"breadcrumbCategories,omitempty"`
	BreadcrumbLevels           []string `json:"breadcrumbLevels,omitempty" yaml:"breadcrumbLevels,omitempty"`
	BreadcrumbsTypes           []string `json:"breadcrumbsTypes,omitempty" yaml:"breadcrumbsTypes,omitempty"`
	BreadcrumbMessages         []string `json:"breadcrumbMessages,omitempty" yaml:"breadcrumbMessages,omitempty"`
	Measurements               []string `json:"measurements,omitempty" yaml:"measurements,omitempty"`
	Operations                 []string `json:"operations,omitempty" yaml:"operations,omitempty"`
}

func (t *TransactionJob) UnmarshalJSON(b []byte) error {
	if t == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw transactionJobRaw
	var err error
	if err = json.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(t)
}

func (t TransactionJob) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.intoRaw())
}

func (t *TransactionJob) UnmarshalYaml(b []byte) error {
	if t == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw transactionJobRaw

	var err error
	if err = yaml.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(t)
}

func (t TransactionJob) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(t.intoRaw())
}

func (t TransactionJob) intoRaw() transactionJobRaw {
	return transactionJobRaw{
		TransactionDurationMax:     t.TransactionDurationMax.String(),
		TransactionDurationMin:     t.TransactionDurationMin.String(),
		TransactionTimestampSpread: t.TransactionTimestampSpread.String(),
		MinSpans:                   t.MinSpans,
		MaxSpans:                   t.MaxSpans,
		NumReleases:                t.NumReleases,
		NumUsers:                   t.NumUsers,
		MinBreadcrumbs:             t.MinBreadcrumbs,
		MaxBreadcrumbs:             t.MaxBreadcrumbs,
		BreadcrumbCategories:       t.BreadcrumbCategories,
		BreadcrumbLevels:           t.BreadcrumbLevels,
		BreadcrumbsTypes:           t.BreadcrumbsTypes,
		BreadcrumbMessages:         t.BreadcrumbMessages,
		Measurements:               t.Measurements,
		Operations:                 t.Operations,
	}
}

func (raw transactionJobRaw) into(result *TransactionJob) error {
	var err error

	if result == nil {
		return errors.New("into called with nil result")
	}

	var transactionDurationMax time.Duration
	if len(raw.TransactionDurationMax) > 0 {
		transactionDurationMax, err = time.ParseDuration(raw.TransactionDurationMax)
	}
	if err != nil {
		return fmt.Errorf("deserialization error, invalid duration %s passed to transactionDurationMax", raw.TransactionDurationMax)
	}

	var transactionDurationMin time.Duration
	if len(raw.TransactionDurationMin) > 0 {
		transactionDurationMin, err = time.ParseDuration(raw.TransactionDurationMin)
	}
	if err != nil {
		return fmt.Errorf("deserialization error, invalid duration %s passed to transactionDurationMin", raw.TransactionDurationMin)
	}

	var transactionTimestampSpread time.Duration
	if len(raw.TransactionTimestampSpread) > 0 {
		transactionTimestampSpread, err = time.ParseDuration(raw.TransactionTimestampSpread)
	}
	if err != nil {
		return fmt.Errorf("deserialization error, invalid duration %s passed to transactionTimestampSpread", raw.TransactionTimestampSpread)
	}

	result.TransactionDurationMax = transactionDurationMax
	result.TransactionDurationMin = transactionDurationMin
	result.TransactionTimestampSpread = transactionTimestampSpread
	result.MinSpans = raw.MinSpans
	result.MaxSpans = raw.MaxSpans
	result.NumReleases = raw.NumReleases
	result.NumUsers = raw.NumUsers
	result.MinBreadcrumbs = raw.MinBreadcrumbs
	result.MaxBreadcrumbs = raw.MaxBreadcrumbs
	result.BreadcrumbCategories = raw.BreadcrumbCategories
	result.BreadcrumbLevels = raw.BreadcrumbLevels
	result.BreadcrumbsTypes = raw.BreadcrumbsTypes
	result.BreadcrumbMessages = raw.BreadcrumbMessages
	result.Measurements = raw.Measurements
	result.Operations = raw.Operations
	return nil
}

func init() {
	RegisterTestType("transaction", newTransactionLoadTester, nil)
}
