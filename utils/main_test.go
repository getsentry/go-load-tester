package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJsonSerialisation(t *testing.T) {
	uuidString := "1f94e1f8-98c2-11ec-b909-0242ac120002"
	id, err := uuid.Parse(uuidString)
	if err != nil {
		t.Error(err)
	}
	if val := UuidAsHex(id); val != "1f94e1f898c211ecb9090242ac120002" {
		t.Errorf("UUidAsHex unexpected value %s", val)
	}
}

func TestExponentialBackoff(t *testing.T) {
	initial, _ := time.ParseDuration("5s")
	max, _ := time.ParseDuration("5m")

	// create a sqrt(2) exponential backoff
	backoff := ExponentialBackoff(initial, max, 1.4143)

	allExpected := []int{5, 7, 10, 14, 20, 28, 40, 56, 80, 113, 160, 226, 300, 300, 300}
	for _, expected := range allExpected {
		if actual := int(backoff().Seconds()); expected != actual {
			t.Errorf("Exponential backoff expected %d got %d", expected, actual)
		}
	}
}
