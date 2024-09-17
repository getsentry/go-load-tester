package dataproviders

import (
	"fmt"
	"testing"
	"time"
)

func TestConstant(t *testing.T) {
	var constant Value
	constant = NewConst(10)
	if constant.GetValue(1) != 10 {
		t.Error("Invalid value returned")
	}
}

func TestSequence(t *testing.T) {
	var seq = Sequence{
		From: 4,
		Step: 2,
	}
	val := seq.GetValue(2)
	if val != uint64(8) {
		t.Error(fmt.Printf("Invalid value returned %d", val))
	}
}

func TestRandomSet(t *testing.T) {
	set, _ := NewRandomSetFromConfig(map[string]interface{}{
		"alphabet": []interface{}{"a", "b", "c"},
	})
	value := set.GetValue(1)
	if !(value == "a" || value == "b" || value == "c") {
		t.Error(fmt.Printf("Missing value %s", value))
	}
}

func TestTimeStamp(t *testing.T) {
	ts, _ := NewTimestampFromConfig(map[string]interface{}{
		"format": "2006-01-02T15:04:05",
	})
	ts.GetValue(1)
}

func TestRandomTimestamp(t *testing.T) {
    config := map[string]interface{}{
        "start":  "2023-01-01T00:00:00",
        "end":    "2023-12-31T23:59:59",
        "format": "2006-01-02T15:04:05",
    }

    randomTimestamp, err := NewRandomTimestampFromConfig(config)
    if err != nil {
        t.Fatalf("Error initializing RandomTimestamp: %v", err)
    }

    randomTimeStr := randomTimestamp.GetValue().(string)
    randomTime, err := time.Parse("2006-01-02T15:04:05", randomTimeStr)
    if err != nil {
        t.Fatalf("Error parsing random timestamp: %v", err)
    }

    startTime, err := time.Parse("2006-01-02T15:04:05", config["start"].(string))
    if err != nil {
        t.Fatalf("Error parsing start time: %v", err)
    }
    endTime, err := time.Parse("2006-01-02T15:04:05", config["end"].(string))
    if err != nil {
        t.Fatalf("Error parsing end time: %v", err)
    }

    if randomTime.Before(startTime) || randomTime.After(endTime) {
        t.Errorf("Generated timestamp %v is out of range [%v, %v]", randomTime, startTime, endTime)
    }
}

func TestRandomInt(t *testing.T) {
	ts, _ := NewRandomIntegerFromConfig(map[string]interface{}{
		"min": 5.0,
		"max": 5.0,
	})
	val := ts.GetValue(1)
	if val != 5 {
		t.Error(fmt.Printf("Missing value %d", val))
	}
}

func TestRandomString(t *testing.T) {
	val, _ := NewRandomStringFromConfig(map[string]interface{}{
		"minSize": 5.0,
		"maxSize": 5.0,
	})

	v := val.GetValue(1)
	length := len(v.(string))
	if length != 5 {
		t.Error(fmt.Printf("Wrong length %d", length))
	}
}
