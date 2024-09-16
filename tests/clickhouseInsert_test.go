package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/getsentry/go-load-tester/tests/dataproviders"
)

func TestSplitter(t *testing.T) {
	test_param := TestParams{
		Name:           "my_test",
		Description:    "my_desc",
		TestType:       "clickhouseInsert",
		AttackDuration: 3600,
		NumMessages:    3600,
		Per:            1,
		Params: json.RawMessage(`
		{ 
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
				}           
			}
		}
		`),
	}

	result, err := clickhouseInsertLoadSplitter(test_param, 2)
	if err != nil {
		t.Error(fmt.Printf("split returned error %s", err))
	}

	if len(result) != 2 {
		t.Error(fmt.Printf("Split returned wrong number of results %d", len(result)))
	}

	result1 := result[0]
	if result1.Per != 2 {
		t.Error(fmt.Printf("Split did not increase the Per. Actual value %d", result1.Per))
	}

	var jsonClickhouseQueryParams dataproviders.ClickhouseInsertJobRaw
	json.Unmarshal(result1.Params, &jsonClickhouseQueryParams)
	if jsonClickhouseQueryParams.Partitions != 2 {
		t.Error(fmt.Printf("Partitions value did not increase. Actual value %d", jsonClickhouseQueryParams.Partitions))
	}

	result2 := result[1]
	json.Unmarshal(result2.Params, &jsonClickhouseQueryParams)
	if jsonClickhouseQueryParams.PartitionId != 1 {
		t.Error(fmt.Printf("Partition id value is wrong. Actual value %d", jsonClickhouseQueryParams.PartitionId))
	}
}
