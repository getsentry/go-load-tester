package tests

import (
	"regexp"
	"testing"
)

func TestVersionGenerator(t *testing.T) {
	type test struct {
		numSegments uint64
		maxVal      uint64
		pattern     string
	}

	var tests = []test{
		{1, 9, `^\d$`},
		{1, 999, `^\d{1,3}$`},
		{4, 9, `^\d\.\d\.\d\.\d$`},
		{4, 255, `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`},
	}

	// since we are generating random values run the test a few times (not ideal)
	for i := 0; i < 10; i++ {
		for _, test := range tests {
			actual := VersionGenerator(test.numSegments, test.maxVal)()
			matched, err := regexp.MatchString(test.pattern, actual)
			if !matched || err != nil {
				t.Errorf("failed to match %s for VersionGenerator(%d,%d)", actual, test.numSegments, test.maxVal)
			}
		}
	}
}

func TestTransactionGenerator(T *testing.T) {

}
