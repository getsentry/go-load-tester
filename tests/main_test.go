package tests

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestUnmarshalRequest(t *testing.T) {
	var request = `{
  "attackDuration": "10s",
  "description": "description",
  "labels": [["l1", "v1"], ["l2", "v2"]],
  "name": "name",
  "numMessages": 2,
  "params": {"p1":"v1"},
  "per": "1s",
  "testType": "session"
}`
	var v TestParams

	err := json.Unmarshal([]byte(request), &v)

	if err != nil {
		t.Errorf("failed to unmarshal request %v", err)
	}

	expectedValue := TestParams{
		AttackDuration: time.Second * 10,
		Description:    "description",
		Labels:         [][]string{{"l1", "v1"}, {"l2", "v2"}},
		Name:           "name",
		NumMessages:    2,
		Params:         []byte(`{"p1":"v1"}`),
		Per:            time.Second,
		TestType:       "session",
	}

	if !reflect.DeepEqual(v, expectedValue) {
		t.Errorf("error deserializing testParams:\n expected:%+v\n  got:%+v", expectedValue, v)
	}
}
