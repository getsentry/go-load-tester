package utils

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

type testStruct struct {
	A StringDuration `json:"a" yaml:"a"`
}

func TestJSONSerialization(t *testing.T) {
	var x testStruct = testStruct{
		A: StringDuration(2 * time.Second),
	}

	result, err := json.Marshal(x)

	if err != nil {
		t.Errorf("Failed to serialize struct with StringDuration error=%s", err)
	}
	stringResult := string(result)
	expected := `{"a":"2s"}`

	if stringResult != expected {
		t.Errorf("Failed deserialzation expected:%s got %s", expected, stringResult)
	}
}

func TestJSONDeserialization(t *testing.T) {
	raw := `{"a":"2s"}`
	var x testStruct

	err := json.Unmarshal([]byte(raw), &x)

	if err != nil {
		t.Errorf("failed to deserialize test structure. error=%s", err)
	}

	if time.Duration(x.A) != 2*time.Second {
		t.Errorf("expected %s got %v", 2*time.Second, x.A)
	}
}

func TestYAMLSerialization(t *testing.T) {
	var x testStruct = testStruct{
		A: StringDuration(2 * time.Second),
	}

	result, err := yaml.Marshal(x)

	if err != nil {
		t.Errorf("Failed to serialize struct with StringDuration error=%s", err)
	}
	stringResult := string(result)
	expected := "a: 2s\n"

	if stringResult != expected {
		t.Errorf("Failed deserialzation expected:%s got %s", expected, stringResult)
	}
}

func TestYAMLDeserialization(t *testing.T) {
	raw := "a: 2s"

	var x testStruct

	err := yaml.Unmarshal([]byte(raw), &x)

	if err != nil {
		t.Errorf("failed to deserialize test structure. error=%s", err)
	}

	if time.Duration(x.A) != 2*time.Second {
		t.Errorf("expected %s got %v", 2*time.Second, x.A)
	}
}
