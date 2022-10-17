package tests

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"

	"github.com/getsentry/go-load-tester/utils"
)

var transactionJob = TransactionJob{
	NumProjects:                1,
	TransactionTimestampSpread: utils.StringDuration(3 * time.Minute),
	TransactionJobCommon: TransactionJobCommon{
		TransactionDurationMin: utils.StringDuration(time.Minute),
		TransactionDurationMax: utils.StringDuration(2 * time.Minute),
		MinSpans:               4,
		MaxSpans:               5,
		NumReleases:            6,
		NumUsers:               7,
		MinBreadcrumbs:         8,
		MaxBreadcrumbs:         9,
		BreadcrumbCategories:   []string{"a1", "a2"},
		BreadcrumbLevels:       []string{"b1", "b2"},
		BreadcrumbsTypes:       []string{"c1", "c2"},
		BreadcrumbMessages:     []string{"d1", "d2"},
		Measurements:           []string{"e1", "e2"},
		Operations:             []string{"f1", "f2"},
	},
}

var transactionJobRawJSON = `
{
	"numProjects":1,
	"transactionTimestampSpread":"3m",
	"transactionDurationMax":"2m",
	"transactionDurationMin":"1m",
	"minSpans":4,
	"maxSpans":5,
	"numReleases":6,
	"numUsers":7,
	"minBreadcrumbs":8,
	"maxBreadcrumbs":9,
	"breadcrumbCategories":["a1","a2"],
	"breadcrumbLevels":["b1","b2"],
	"breadcrumbsTypes":["c1","c2"],
	"breadcrumbMessages":["d1","d2"],
	"measurements":["e1","e2"],
	"operations":["f1","f2"]
}
`

var transactionJobRawYAML = `
numProjects: 1
transactionTimestampSpread: 3m0s
transactionDurationMax: 2m0s
transactionDurationMin: 1m0s
minSpans: 4
maxSpans: 5
numReleases: 6
numUsers: 7
minBreadcrumbs: 8
maxBreadcrumbs: 9
breadcrumbCategories:
- a1
- a2
breadcrumbLevels:
- b1
- b2
breadcrumbsTypes:
- c1
- c2
breadcrumbMessages:
- d1
- d2
measurements:
- e1
- e2
operations:
- f1
- f2
`

var transactionJobV2 = TransactionJobV2{
	ProjectDistribution: []ProjectProfile{
		{
			NumProjects:        100,
			RelativeFreqWeight: 1.0,
			TimestampHistogram: []TimestampHistogramBucket{
				{
					Weight:   5.0,
					MaxDelay: utils.StringDuration(time.Second),
				},
			},
		},
	},
	TransactionJobCommon: TransactionJobCommon{
		TransactionDurationMin: utils.StringDuration(time.Minute),
		TransactionDurationMax: utils.StringDuration(2 * time.Minute),
		MinSpans:               4,
		MaxSpans:               5,
		NumReleases:            6,
		NumUsers:               7,
		MinBreadcrumbs:         8,
		MaxBreadcrumbs:         9,
		BreadcrumbCategories:   []string{"a1", "a2"},
		BreadcrumbLevels:       []string{"b1", "b2"},
		BreadcrumbsTypes:       []string{"c1", "c2"},
		BreadcrumbMessages:     []string{"d1", "d2"},
		Measurements:           []string{"e1", "e2"},
		Operations:             []string{"f1", "f2"},
	},
}

var transactionJobV2RawJSON = `
{
	"projectDistribution": [
	  {
		"numProjects": 100,
		"relativeFreqWeight" : 1.0,
		"timestampHistogram": [
		  { "weight": 5.0, "maxDelay": "1s"}
		]
	  }
	],
	"transactionDurationMax":"2m",
	"transactionDurationMin":"1m",
	"minSpans":4,
	"maxSpans":5,
	"numReleases":6,
	"numUsers":7,
	"minBreadcrumbs":8,
	"maxBreadcrumbs":9,
	"breadcrumbCategories":["a1","a2"],
	"breadcrumbLevels":["b1","b2"],
	"breadcrumbsTypes":["c1","c2"],
	"breadcrumbMessages":["d1","d2"],
	"measurements":["e1","e2"],
	"operations":["f1","f2"]
}
`

var transactionJobV2RawYAML = `
projectDistribution:
- numProjects: 100
  relativeFreqWeight: 1
  TimestampHistogram:
  - weight: 5
    maxDelay: 1s
transactionDurationMax: 2m0s
transactionDurationMin: 1m0s
minSpans: 4
maxSpans: 5
numReleases: 6
numUsers: 7
minBreadcrumbs: 8
maxBreadcrumbs: 9
breadcrumbCategories:
- a1
- a2
breadcrumbLevels:
- b1
- b2
breadcrumbsTypes:
- c1
- c2
breadcrumbMessages:
- d1
- d2
measurements:
- e1
- e2
operations:
- f1
- f2
`

func TestTransactionJsonSerialization(t *testing.T) {
	// do a round trip and compare we end up in the same place
	data, err := json.Marshal(&transactionJob)
	if err != nil {
		t.Error("Could not serialize transactionJob job to JSON")
		return
	}
	var result TransactionJob
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize transactionJob")
		return
	}

	if diff := cmp.Diff(transactionJob, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJob JSON serialisation round trip (-expect +actual)\n %s", diff)
	}
}

func TestTransactionJsonDeserialization(t *testing.T) {
	var result TransactionJob
	err := json.Unmarshal([]byte(transactionJobRawJSON), &result)
	if err != nil {
		t.Error("Could not deserialize transactionJob")
		return
	}
	if diff := cmp.Diff(transactionJob, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJob JSON serialisation round trip (-expect +actual)\n %s", diff)
		return
	}
}

func TestTransactionYamlSerialisation(t *testing.T) {
	data, err := yaml.Marshal(&transactionJob)
	if err != nil {
		t.Error("Could not serialize transactionJob job to YAML")
		return

	}
	var result TransactionJob
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize transactionJob")
		return
	}

	if diff := cmp.Diff(transactionJob, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJob YAML serialisation round trip (-expect +actual)\n %s", diff)
	}
}

func TestTransactionYamlDeserialization(t *testing.T) {
	var result TransactionJob
	err := yaml.Unmarshal([]byte(transactionJobRawYAML), &result)
	if err != nil {
		t.Error("Could not deserialize transactionJob")
		return
	}
	if diff := cmp.Diff(transactionJob, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJob YAML serialisation round trip (-expect +actual)\n %s", diff)
	}
}

func TestTransactionV2JsonSerialization(t *testing.T) {
	// do a round trip and compare we end up in the same place
	data, err := json.Marshal(&transactionJobV2)
	if err != nil {
		t.Error("Could not serialize transactionJobV2 job to JSON")
		return
	}
	var result TransactionJobV2
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize transactionJobV2")
		return
	}

	if diff := cmp.Diff(transactionJobV2, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJobV2 JSON serialisation round trip (-expect +actual)\n %s", diff)
	}
}

func TestTransactionV2JsonDeserialization(t *testing.T) {
	var result TransactionJobV2
	err := json.Unmarshal([]byte(transactionJobV2RawJSON), &result)
	if err != nil {
		t.Errorf("Could not deserialize transactionJobV2\n %s", err)
		return
	}
	if diff := cmp.Diff(transactionJobV2, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJobV2 YAML serialisation round trip (-expect +actual)\n %s", diff)
	}
}

func TestTransactionV2YamlSerialisation(t *testing.T) {
	data, err := yaml.Marshal(&transactionJobV2)
	if err != nil {
		t.Error("Could not serialize transactionJobV2 job to YAML")
		return
	}

	var result TransactionJobV2
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize transactionJobV2")
		return
	}

	if diff := cmp.Diff(transactionJobV2, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJob YAML serialisation round trip (-expect +actual)\n %s", diff)
	}
}

func TestTransactionV2YamlDeserialization(t *testing.T) {
	var result TransactionJobV2
	err := yaml.Unmarshal([]byte(transactionJobV2RawYAML), &result)
	if err != nil {
		t.Error("Could not deserialize transactionJobV2")
		return
	}
	if diff := cmp.Diff(transactionJobV2, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJobV2 YAML serialisation round trip (-expect +actual)\n %s", diff)
	}
}

func TestTimespreadGenerator(t *testing.T) {
	var profiles []ProjectProfile = []ProjectProfile{
		{
			NumProjects:        1,   // unused in test
			RelativeFreqWeight: 1.0, // unused in test
			TimestampHistogram: []TimestampHistogramBucket{
				{
					Weight:   0, // effectively disable this bucket
					MaxDelay: utils.StringDuration(time.Second),
				},
				{
					Weight:   1,
					MaxDelay: utils.StringDuration(time.Minute),
				},
			},
		},
		{
			NumProjects:        1,   // unused in test
			RelativeFreqWeight: 1.0, // unused in test
			TimestampHistogram: []TimestampHistogramBucket{
				{
					Weight:   1,
					MaxDelay: utils.StringDuration(time.Second),
				},
				{
					Weight:   0, // disable bucket
					MaxDelay: utils.StringDuration(time.Minute),
				},
			},
		},
		{
			NumProjects:        1,   // unused in test
			RelativeFreqWeight: 1.0, // unused in test
			TimestampHistogram: []TimestampHistogramBucket{
				{
					Weight:   0,
					MaxDelay: utils.StringDuration(time.Second),
				},
				{
					Weight:   1, // disable bucket
					MaxDelay: utils.StringDuration(time.Minute),
				},
				{
					Weight:   0, // disable bucket
					MaxDelay: utils.StringDuration(2 * time.Minute),
				},
			},
		},
	}

	testCases := []struct {
		profileIdx int
		timeMin    time.Duration
		timeMax    time.Duration
	}{
		{profileIdx: 0, timeMin: time.Second, timeMax: time.Minute},
		{profileIdx: 1, timeMin: 0, timeMax: time.Second},
		{profileIdx: 2, timeMin: time.Second, timeMax: time.Minute},
	}

	generator := timeSpreadGenerator(profiles)
	for _, testCase := range testCases {
		for idx := 0; idx < 10; idx++ {
			timestamp := generator(testCase.profileIdx)

			if timestamp < testCase.timeMin || timestamp > testCase.timeMax {
				t.Errorf("failed to generate timespread in specified interval got %s expected values in [%s,%s]",
					timestamp.String(), testCase.timeMin, testCase.timeMax)
				return
			}
		}
	}
}

// This test is more a sanity test, it is not strictly guaranteed to work since a random distribution
// may not spread the results as expected but for about 1,000,000 samples we should get an error of less than 0.01,
// in rare occasions may theoretically fail.
func TestTimespreadGeneratorAporxEqual(t *testing.T) {
	const NumInvocations = 1_000_000
	var profiles []ProjectProfile = []ProjectProfile{
		{
			NumProjects:        1,   // unused in test
			RelativeFreqWeight: 1.0, // unused in test
			TimestampHistogram: []TimestampHistogramBucket{
				{
					Weight:   10, // effectively disable this bucket
					MaxDelay: utils.StringDuration(time.Second),
				},
				{
					Weight:   5,
					MaxDelay: utils.StringDuration(time.Minute),
				},
				{
					Weight:   1,
					MaxDelay: utils.StringDuration(time.Hour),
				},
				{
					Weight:   0.1,
					MaxDelay: utils.StringDuration(2 * time.Hour),
				},
			},
		},
	}

	generator := timeSpreadGenerator(profiles)
	histograms := profiles[0].TimestampHistogram
	numHistograms := len(histograms)
	counters := make([]float64, numHistograms)
	for idx := 0; idx < NumInvocations; idx++ {
		timestamp := generator(0)
		for histIdx := 0; histIdx < numHistograms; histIdx++ {
			maxDelay := time.Duration(histograms[histIdx].MaxDelay)
			if timestamp <= maxDelay {
				counters[histIdx]++
				break
			}
		}
	}
	var cumulatedWeight float64
	for histIdx := 0; histIdx < numHistograms; histIdx++ {
		cumulatedWeight += histograms[histIdx].Weight
	}

	// muliplying it by cummulatedWeight/Weight*NumSamples should get us to around 1
	for histIdx := 0; histIdx < numHistograms; histIdx++ {
		adjustedBucket := (counters[histIdx] * cumulatedWeight) / (histograms[histIdx].Weight * NumInvocations)
		if !isAboutOne(adjustedBucket, 0.01) {
			t.Errorf("Expected the adjusted bucket to be about 1 but got %f", adjustedBucket)
		}
	}

}

func isAboutOne(val float64, error float64) bool {
	errorMargin := math.Abs(val * error)
	return val-errorMargin < 1 && val+errorMargin > 1
}

func TestTransactionGeneration(t *testing.T) {
	var tc = transactionJob.TransactionJobCommon

	generator := TransactionGenerator(tc)

	tr := generator(5 * time.Second)

	if !isID(tr.EventId) {
		t.Error("invalid eventID")
	}

	timestamp, err := FromUTCString(tr.Timestamp)
	if err != nil {
		t.Errorf("invalid timestamp %s", tr.Timestamp)
	}
	startTimestamp, err := FromUTCString(tr.StartTimestamp)
	if err != nil {
		t.Errorf("invalid startTimestamp %s", tr.StartTimestamp)
	}

	transactionRange := time.Duration(tc.TransactionDurationMax - tc.TransactionDurationMin)

	if startTimestamp.Before(timestamp.Add(-transactionRange)) || startTimestamp.After(timestamp) {
		t.Error("Bad start timestamp")
	}

	numBreadcurmbs := uint64(len(tr.Breadcrumbs))

	if numBreadcurmbs < tc.MinBreadcrumbs || numBreadcurmbs > tc.MaxBreadcrumbs {
		t.Errorf("Bad number of breadcrumbs %d not in [%d,%d]]", numBreadcurmbs, tc.MinBreadcrumbs, tc.MaxBreadcrumbs)
	}

	numSpans := uint64(len(tr.Spans))

	if numSpans < tc.MinSpans || numSpans > tc.MaxSpans {
		t.Errorf("Bad number of breadcrumbs %d not in [%d,%d]]", numBreadcurmbs, tc.MinBreadcrumbs, tc.MaxBreadcrumbs)
	}
}

func isID(s string) bool {
	return len(s) == 32
}
