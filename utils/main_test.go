package utils

import (
	"github.com/google/go-cmp/cmp"
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

func TestRandomChoiceDoseNotPanic(t *testing.T) {

	type test struct {
		name        string
		weights     []int64
		choices     *[]string
		expectError bool
	}
	var choices []string = []string{"a", "b", "c"}
	var tests []test = []test{
		{name: "more choices", weights: []int64{}, choices: &choices, expectError: false},
		{name: "less choices", weights: []int64{1, 2, 3, 4}, choices: &choices, expectError: false},
		{name: "0 weights", weights: []int64{0, 0, 0}, choices: &choices, expectError: true},
		{name: "less 0 weights", weights: []int64{0}, choices: &choices, expectError: false},
		{name: "more 0 weights", weights: []int64{0, 0, 0, 0}, choices: &choices, expectError: true},
		{name: "no choices", weights: []int64{1, 2, 3}, choices: &[]string{}, expectError: true},
	}

	for _, test := range tests {
		_, err := RandomChoice(*test.choices, test.weights)
		if (err != nil) != test.expectError {
			t.Errorf("test: %s failed", test.name)
		}
	}
}

func TestPerSecond(t *testing.T) {
	type test struct {
		name     string
		elements int64
		interval time.Duration
		want     float64
	}

	const acceptableError = 0.01
	const lowerBound = 1 - acceptableError
	const upperBound = 1 + acceptableError

	var tests []test = []test{
		{"one/sec", 1, time.Second, 1.0},
		{"one/3sec", 1, time.Second * 3, 1.0 / 3},
		{"7/h", 7, time.Hour, 7.0 / 3600},
		{"3/ms", 3, time.Millisecond, 3.0 / 0.001},
	}

	for _, test := range tests {
		var got, err = PerSecond(test.elements, test.interval)
		if err != nil {
			t.Errorf("Test: %s faile with %v", test.name, err)
		}

		if got < test.want*lowerBound || got > test.want*upperBound {
			t.Errorf("Expecting %f got %f", test.want, got)
		}
	}
}

func TestDivide(t *testing.T) {

	type testData struct {
		numerator   int
		denominator int
		expected    []int
	}

	var tests = []testData{
		{numerator: 1, denominator: 1, expected: []int{1}},
		{numerator: 5, denominator: 3, expected: []int{2, 2, 1}},
		{numerator: 6, denominator: 3, expected: []int{2, 2, 2}},
		{numerator: 3, denominator: 5, expected: []int{1, 1, 1, 0, 0}},
		{numerator: -5, denominator: 3, expected: []int{-2, -2, -1}},
		{numerator: -6, denominator: 3, expected: []int{-2, -2, -2}},
		{numerator: 0, denominator: 3, expected: []int{0, 0, 0}},
	}

	for _, test := range tests {
		result, err := Divide(test.numerator, test.denominator)
		if err != nil {
			t.Errorf("Divide(%d, %d) caused error:\n%v", test.numerator, test.denominator, err)
		}
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("Failed to serialize, session JSON serialisation round trip (-expect +actual)\n %s", diff)
		}
	}
}
