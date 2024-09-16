package dataproviders

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBasicMap(t *testing.T) {
	var a Value
	a = NewConst("bla")
	b := NewConst(10)

	structValue := NewStructValue(map[string]Value{
		"constStr": a,
		"constInt": b,
		"seq":      &Sequence{},
	}, []StructValue{})

	val := structValue.GetValue(1)
	compare := map[string]interface{}{
		"constStr": "bla",
		"constInt": 10,
		"seq":      1,
	}
	if val["constStr"] != compare["constStr"] {
		t.Error("Invalid value returned")
	}
	if val["constInt"] != compare["constInt"] {
		t.Error("Invalid value returned")
	}
	if val["sequence"] != compare["sequence"] {
		t.Error(fmt.Printf("Invalid value returned %s %s", val["seq"], compare["seq"]))
	}
}

func TestBatch(t *testing.T) {
	builder := NewBatchBuilder(
		*NewStructValue(
			map[string]Value{
				"constStr": NewConst("bla"),
				"seq":      &Sequence{Step: 1},
			},
			[]StructValue{},
		),
		3,
	)

	batch1 := builder.BuildBatch()
	expected := [3]map[string]interface{}{
		{"constStr": "bla", "seq": uint64(0)},
		{"constStr": "bla", "seq": uint64(1)},
		{"constStr": "bla", "seq": uint64(2)},
	}
	for i, val := range batch1 {
		if !reflect.DeepEqual(val, expected[i]) {
			t.Error(fmt.Printf("Batch 1 does not match %s %s", batch1, expected))
		}
	}

	batch2 := builder.BuildBatch()
	expected2 := [3]map[string]interface{}{
		{"constStr": "bla", "seq": uint64(3)},
		{"constStr": "bla", "seq": uint64(4)},
		{"constStr": "bla", "seq": uint64(5)},
	}
	for i, val := range batch2 {
		if !reflect.DeepEqual(val, expected2[i]) {
			t.Error(fmt.Printf("Batch 2 does not match %s", batch2))
		}
	}
}
