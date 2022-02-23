package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

var session SessionJob = SessionJob{
	StartedRange:    time.Duration(1 * time.Minute),
	DurationRange:   time.Duration(2 * time.Minute),
	NumReleases:     3,
	NumEnvironments: 4,
	NumUsers:        5,
	OkWeight:        6,
	ExitedWeight:    7,
	ErroredWeight:   8,
	CrashedWeight:   9,
	AbnormalWeight:  10,
}

func TestJsonSerialisation(t *testing.T) {
	data, err := json.Marshal(&session)
	if err != nil {
		t.Error("Could not serialize session to JSON")
	}
	var result SessionJob
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize session")
	}
	if diff := cmp.Diff(session, result); diff != "" {
		t.Errorf("Failed to session JSON serialisation round trip (-expect +actual)\n %s", diff)
	}
}
func TestYamlSerialisation(t *testing.T) {
	data, err := yaml.Marshal(&session)
	if err != nil {
		t.Error("Could not serialize session to JSON")
	}
	var result SessionJob
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		t.Error("Could not deserialize session")
	}
	if diff := cmp.Diff(session, result); diff != "" {
		t.Errorf("Failed to session JSON serialisation round trip (-expect +actual)\n %s", diff)
	}
}
