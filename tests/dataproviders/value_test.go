package dataproviders

import (
	"fmt"
	"testing"
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
