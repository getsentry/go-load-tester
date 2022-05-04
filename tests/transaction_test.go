package tests

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
	"testing"
	"time"
)

var transactionJob = TransactionJob{
	TransactionDurationMax:     time.Minute,
	TransactionDurationMin:     2 * time.Minute,
	TransactionTimestampSpread: 3 * time.Minute,
	MinSpans:                   4,
	MaxSpans:                   5,
	NumReleases:                6,
	NumUsers:                   7,
	MinBreadcrumbs:             8,
	MaxBreadcrumbs:             9,
	BreadcrumbCategories:       []string{"a1", "a2"},
	BreadcrumbLevels:           []string{"b1", "b2"},
	BreadcrumbsTypes:           []string{"c1", "c2"},
	BreadcrumbMessages:         []string{"d1", "d2"},
	Measurements:               []string{"e1", "e2"},
	Operations:                 []string{"f1", "f2"},
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
