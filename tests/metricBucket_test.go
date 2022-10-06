package tests

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestRandomFloatArray(t *testing.T) {
	minElements := 5
	maxElements := 10
	val := randomFloat64Array(minElements, maxElements)

	if len(val) < minElements {
		t.Errorf("Invalid number of elements, expected min %d got %d", minElements, len(val))
		return
	}
	if len(val) > maxElements {
		t.Errorf("Invalid number of elements, expected max %d got %d", maxElements, len(val))
		return
	}

	if !sort.Float64sAreSorted(val) {
		t.Errorf("Array should be sorted and it's not")
		return
	}
}

func TestRandomIntArray(t *testing.T) {
	minElements := 5
	maxElements := 10
	val := randomIntArray(minElements, maxElements)

	length := len(val)

	if length < minElements {
		t.Errorf("Invalid number of elements, expected min %d got %d", minElements, len(val))
		return
	}
	if length > maxElements {
		t.Errorf("Invalid number of elements, expected max %d got %d", maxElements, len(val))
		return
	}

	var current int32 = 0
	for idx := 0; idx < length; idx++ {
		if val[idx] < current {
			t.Errorf("Array should be sorted and it's not")
			current = val[idx]
			return
		}
	}
}

func TestRandomTags(t *testing.T) {
	numTags := 7
	numVals := 5

	tags := randomTags(numTags, numVals)

	for idx := 1; idx <= numTags; idx++ {
		key := fmt.Sprintf("t%d", idx)
		val, ok := tags[key]

		if !ok {
			t.Errorf("expected key %s not found ", key)
		}

		if !strings.HasPrefix(val, "v") {
			t.Errorf("Invalid prefix in value: %s", val)
		}
		val = val[1:]
		vNum, err := strconv.Atoi(val)

		if err != nil {
			t.Errorf("Invalid postfix for value: %s, expected some integer", val)
			return
		}

		if vNum < 1 || vNum > numVals {
			t.Errorf("Value out of bound expected number to be in [1,%d] got %d", numVals, vNum)
		}
	}
}
