package tests

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJson(t *testing.T) {
	jsonData := json.RawMessage(`{"multiplier": 100}`)

	loadTester := newClickhouseQueryLoadTester("http://localhost:9000", jsonData)

	if _, ok := loadTester.(*clickhouseQueryLoadTester); ok {

	} else {
		t.Error("Invalid type genrated")
	}

}
