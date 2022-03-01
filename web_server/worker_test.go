package web_server

import (
	"encoding/json"
	"testing"
)

func TestCreateRegistrationBody(t *testing.T) {
	testUrl := "10.10.20.3:8088"
	data, err := createRegistrationBody(testUrl)

	if err != nil {
		t.Error(err)
	}
	if data == nil {
		t.Errorf("createRegistrationBody returned nil body")
	}

	var actual struct {
		WorkerUrl string `json:"workerUrl"`
	}

	err = json.Unmarshal(data.Bytes(), &actual)

	if err != nil {
		t.Error(err)
	}

	if actual.WorkerUrl != testUrl {
		t.Errorf("Deserialisation error expecting '%s' got '%s' \n", testUrl, actual.WorkerUrl)
	}

}
