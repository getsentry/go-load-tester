package dataproviders

import "fmt"

type ClickhouseFieldRaw struct {
	ValueType string                 `json:"valueType"`
	Config    map[string]interface{} `json:"config"`
}

type ClickhouseInsertJobRaw struct {
	BatchSize   int64                         `json:"batchSize"`
	TableName   string                        `json:"tableName"`
	Partitions  int                           `json:"partitions"`
	PartitionId int                           `json:"partitionId"`
	Config      map[string]ClickhouseFieldRaw `json:"config"`
}

func (raw *ClickhouseInsertJobRaw) Into(result *ClickhouseInsertJob) error {
	result.BatchSize = raw.BatchSize
	result.TableName = raw.TableName
	result.PartitionId = raw.PartitionId
	if raw.Partitions == 0 {
		result.Partitions = 1
	} else {
		result.Partitions = raw.Partitions
	}
	schema, err := NewStructFromConfig(raw.Config, result.Partitions, result.PartitionId)
	if err != nil {
		return err
	}
	result.Schema = *schema

	return nil
}

type ClickhouseInsertJob struct {
	// Size of each batch to build
	BatchSize int64

	TableName string

	PartitionId int
	Partitions  int

	Schema StructValue
}

func NewStructFromConfig(
	config map[string]ClickhouseFieldRaw,
	partitions int,
	partition_id int,
) (*StructValue, error) {
	ret := make(map[string]Value)
	for key, config := range config {
		val, err := BuildField(config, partitions, partition_id)
		if err != nil {
			return nil, fmt.Errorf("failed init %s, %v", key, err)
		}
		ret[key] = val
	}

	return NewStructValue(ret, []StructValue{}), nil
}

func BuildField(config ClickhouseFieldRaw, partitions int, partition_id int) (Value, error) {
	value_type := config.ValueType

	value_config := config.Config

	switch value_type {
	case "const":
		value, err := NewConstFromConfig(value_config)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "partitionedSequence":
		return &Sequence{
			From: uint64(partition_id),
			Step: uint64(partitions),
		}, nil

	case "sequence":
		return &Sequence{
			From: 0,
			Step: 1,
		}, nil

	case "randomSet":
		value, err := NewRandomSetFromConfig(value_config)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "sequenceSet":
		value, err := NewSequenceSetFromConfig(value_config)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "timestamp":
		value, err := NewTimestampFromConfig(value_config)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "uuid":
		return &UUIDGenerator{}, nil

	case "randomInt":
		value, err := NewRandomIntegerFromConfig(value_config)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "randomFloat":
		value, err := NewRandomFloatFromConfig(value_config)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "randomString":
		value, err := NewRandomStringFromConfig(value_config)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "randomArray":
		valueProvider, err := BuildValueProvider(
			value_config["valueProvider"].(map[string](interface{})),
			partitions,
			partition_id,
		)
		if err != nil {
			return nil, err
		}

		value, err := NewRandomArrayFromConfig(value_config, valueProvider)
		if err != nil {
			return nil, err
		}
		return value, nil

	case "randomMap":
		keyProvider, err := BuildValueProvider(
			value_config["keyProvider"].(map[string](interface{})),
			partitions,
			partition_id,
		)
		if err != nil {
			return nil, err
		}

		valueProvider, err := BuildValueProvider(
			value_config["valueProvider"].(map[string](interface{})),
			partitions,
			partition_id,
		)
		if err != nil {
			return nil, err
		}

		value, err := NewRandomMapFromConfig(value_config, keyProvider, valueProvider)
		if err != nil {
			return nil, err
		}
		return value, nil
	}

	return nil, fmt.Errorf("invalid type %s", value_type)
}

func BuildValueProvider(config map[string](interface{}), partitions int, partition_id int) (Value, error) {
	providerConfig := ClickhouseFieldRaw{
		ValueType: config["valueType"].(string),
		Config:    config["config"].(map[string]interface{}),
	}
	valueProvider, err := BuildField(providerConfig, partitions, partition_id)
	if err != nil {
		return nil, err
	}
	return valueProvider, nil
}
