package dataproviders

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBasicConfig(t *testing.T) {
	config := map[string]ClickhouseFieldRaw{
		"field1": ClickhouseFieldRaw{
			ValueType: "const",
			Config: map[string]interface{}{
				"value": "my_val",
			},
		},
		"field2": ClickhouseFieldRaw{
			ValueType: "partitionedSequence",
			Config:    map[string]interface{}{},
		},
		"field3": ClickhouseFieldRaw{
			ValueType: "sequence",
			Config:    map[string]interface{}{},
		},
	}

	structure, err := NewStructFromConfig(config, 4, 1)
	if err != nil {
		t.Error(fmt.Printf("Invalid config %s", err))
	}

	row := structure.GetValue(1)
	if row["field1"] != "my_val" {
		t.Error(fmt.Printf("Invalid value %s", row["field1"]))
	}
	if row["field2"] != uint64(5) {
		t.Error(fmt.Printf("Invalid value %d", row["field2"]))
	}
	if row["field3"] != uint64(1) {
		t.Error(fmt.Printf("Invalid value %d", row["field3"]))
	}
}

func TestUnmarshal(t *testing.T) {
	data := `{ 
        "batchSize": 5,
        "tableName": "test_table_local",
        "config": {
            "seq": {
                "valueType": "partitionedSequence",
                "config": {}
            },
            "constStr": {
                "valueType": "const",
                "config": {
                    "value": "my_val"
                }
            },
			"method": {
                "valueType": "sequenceSet",
                "config": {
                    "alphabet": ["GET", "POST", "DELETE"]
                }
            },
			"manyValues": {
                "valueType": "randomArray",
                "config": {
                    "valueProvider": {
						"valueType": "const",
						"config": {
							"value": "my_val"
						}		
					},
					"maxSize": 5,
					"minSize": 5
					
                }
            },
			"mapValues": {
                "valueType": "randomMap",
                "config": {
                    "keyProvider": {
						"valueType": "const",
						"config": {
							"value": "my_val"
						}		
					},
					"valueProvider": {
						"valueType": "const",
						"config": {
							"value": "my_val"
						}		
					},
					"maxSize": 5,
					"minSize": 5
					
                }
            }          
        }
    }`

	var doc ClickhouseInsertJobRaw
	err := json.Unmarshal([]byte(data), &doc)
	if err != nil {
		t.Error(fmt.Printf("Error unmarshalling JSON: %s", err))
		return
	}

	struct_config, err := NewStructFromConfig(doc.Config, 1, 0)
	if err != nil {
		t.Error(fmt.Printf("Invalid config: %s", err))
		return
	}
	row := struct_config.GetValue(1)
	_, ok := row["manyValues"].([](interface{}))
	if !ok {
		t.Error(fmt.Printf("Invalid array: %v", row))
		return
	}
}
