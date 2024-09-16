package dataproviders

import (
	"sync"
)

type StructValue struct {
	valueBuilders map[string]Value
	// The flattened field is used to create data providers that produce
	// multi column values that are then merged into the results.
	// We may want to create values for columns that are statistically
	// correlated, so they cannot be generated independently. This is
	// meant to solve this problem. Though it is not implemented yet.
	flattened []StructValue
}

func NewStructValue(
	builders map[string]Value,
	flattened []StructValue,
) *StructValue {
	return &StructValue{
		valueBuilders: builders,
		flattened:     flattened,
	}
}

func (structValue *StructValue) GetValue(
	sequence uint64,
) map[string]interface{} {
	ret := make(map[string]interface{})
	for key, generator := range structValue.valueBuilders {
		ret[key] = generator.GetValue(sequence)
	}
	for _, generator := range structValue.flattened {
		for key, subvalue := range generator.GetValue((sequence)) {
			ret[key] = subvalue
		}
	}
	return ret
}

type BatchBuilder struct {
	rowBuilder StructValue
	sequence   uint64
	batchSize  uint64
	lock       sync.Mutex
}

func NewBatchBuilder(rowBuilder StructValue, batchSize uint64) *BatchBuilder {
	return &BatchBuilder{
		rowBuilder: rowBuilder,
		sequence:   0,
		batchSize:  batchSize,
	}
}

func (builder *BatchBuilder) BuildBatch() []map[string]interface{} {
	builder.lock.Lock()
	var start = builder.sequence
	builder.sequence += builder.batchSize
	builder.lock.Unlock()

	var ret []map[string]interface{}
	var i uint64
	for i = 0; i < builder.batchSize; i++ {
		ret = append(ret, builder.rowBuilder.GetValue(start+i))
	}

	return ret
}
