package tests

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalJson(t *testing.T) {
	jsonData := json.RawMessage(`{"multiplsier": 100}`)

	loadTester := newClickhouseQueryLoadTester("http://localhost:9000", jsonData)
	//expected := ClickhouseQueryJob{
	//	multiplier: 100,
	//}

	if _, ok := loadTester.(*clickhouseQueryLoadTester); ok {

	} else {
		t.Error("Invalid type genrated")
	}

}
