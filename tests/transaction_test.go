package tests

import (
	"encoding/json"
	"fmt"
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
		TransactionDurationMax: utils.StringDuration(time.Minute),
		TransactionDurationMin: utils.StringDuration(2 * time.Minute),
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

func TestTransactionGeneration(t *testing.T) {
}

func TestTransactionJsonSerialisation(t *testing.T) {
	data, err := json.Marshal(&transactionJob)
	if err != nil {
		t.Error("Could not serialize transactionJob job to JSON")

	}
	var result TransactionJob
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize transactionJob")
	}

	if diff := cmp.Diff(transactionJob, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJob JSON serialisation round trip (-expect +actual)\n %s", diff)
	}
}
func TestTransactionYamlSerialisation(t *testing.T) {
	data, err := yaml.Marshal(&transactionJob)
	if err != nil {
		t.Error("Could not serialize transactionJob job to JSON")

	}
	var result TransactionJob
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize transactionJob")
	}

	if diff := cmp.Diff(transactionJob, result); diff != "" {
		t.Errorf("Failed to serialize, transactionJob JSON serialisation round trip (-expect +actual)\n %s", diff)
	}
}

type base struct {
	A string
	B int
}

type D2 struct {
	base
	c int
}

func TestIncludedStructs(t *testing.T) {
	var d D2
	var raw = `
{ "a": "a", "b": 1, "c":2 }
`
	err := json.Unmarshal([]byte(raw), &d)

	if err != nil {
		t.Error("failed to unmarshal", err)
	}
	fmt.Printf("Unmarshaled %+v", d)
}
