package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	vegeta "github.com/tsenart/vegeta/lib"
	"gopkg.in/yaml.v2"
)
import "sync"

// TestParams is Implemented by all parameter test classes
//
// This is how commands are passed in http requests
// There are 3 main parts to the params:
//	1. Details about the attack duration and intensity ( AttackDuration, Per, NumMessages)
//  2. Type of the attack, (the Name field) used to dispatch the attack to the appropriate targeter
//  3. Parameters specific to the attack (used by the targeter) Params (structure depends on the targeter used)
//  The Description field is optional and used for documenting the attack (e.g. in reporting)
type TestParams struct {
	Name           string
	Description    string
	TestType       string
	AttackDuration time.Duration // total time of Attack
	NumMessages    int           // number of messages to be sent in Per
	Per            time.Duration // the unit of duration in which to send NumMessages
	Params         json.RawMessage
}

// LoadTesterBuilder is a function that when given a URL and a read channel of
// raw JSON messages returns a LoadTester that is able to change the events it generates
// to reflect the parameter passed through the JSON messages channel.
// Note: the raw JSON messages received through the channel need to be "compatible" with the specific
// targeter. Getting the proper builder for a type of message is outside this function's responsibilities
// (the dispatch is done via GetLoadTester inside the worker)
type LoadTesterBuilder func(url string, params json.RawMessage) LoadTester

// LoadSplitter is a function that knows how to split a load test request between multiple
// workers. In the simplest (and most common) case it just splits the load messages/timeInterval to
// the number of workers by giving each worker the load messages/(timeInterval * numWorkers).
// If this is your case just use SimpleLoadSplitter, if you need something more sophisticated
// implement your own that decomposes your TestParams in the proper way.
// Note: The function must return a slice of TestParams of size numWorkers.
type LoadSplitter func(masterParams TestParams, numWorkers int) ([]TestParams, error)

// LoadTester is an interface implemented by all load tests.
// This is used by the web_server.worker to handle loads based on the TestParams passed in the
// request.
type LoadTester interface {
	// GetTargeter will be called by the worker at the beginning of an attack in order to
	// create a targeter for the particular TestParams passed. This Targeter will be used
	// during the attack to construct requests (this function will be called once per attack)
	GetTargeter() vegeta.Targeter
	// ProcessResult will be called by the worker during an attack for each Result returned by the system
	// under test
	ProcessResult(res *vegeta.Result)
}

func SimpleLoadSplitter(masterParams TestParams, numWorkers int) ([]TestParams, error) {
	if numWorkers <= 0 {
		return nil, fmt.Errorf("invalid number of workers %d need at least 1", numWorkers)
	}
	// divide attack intensity among workers
	newParams := masterParams
	newParams.Per = time.Duration(numWorkers) * masterParams.Per
	retVal := make([]TestParams, 0, numWorkers)
	for idx := 0; idx < numWorkers; idx++ {
		retVal = append(retVal, newParams)
	}
	return retVal, nil
}

func RegisterTestType(name string, tester LoadTesterBuilder, splitter LoadSplitter) {
	testHandlers.lock.Lock()
	defer testHandlers.lock.Unlock()
	testHandlers.loadTesters[name] = tester
	if splitter != nil {
		testHandlers.loadSplitters[name] = splitter
	}
}

// GetLoadTester returns the TargeterBuilder for a particular type of message.
func GetLoadTester(testType string) LoadTesterBuilder {
	testHandlers.lock.Lock()
	defer testHandlers.lock.Unlock()
	return testHandlers.loadTesters[testType]
}

// GetLoadSplitter a loadSplitter for the current type of test.
func GetLoadSplitter(testType string) LoadSplitter {
	testHandlers.lock.Lock()
	defer testHandlers.lock.Unlock()
	if retVal, ok := testHandlers.loadSplitters[testType]; ok && retVal != nil {
		return retVal
	}
	return SimpleLoadSplitter
}

// testHandlers define behaviour for all supported test types
var testHandlers = struct {
	loadTesters   map[string]LoadTesterBuilder
	loadSplitters map[string]LoadSplitter
	lock          sync.Mutex
}{
	loadTesters:   make(map[string]LoadTesterBuilder),
	loadSplitters: make(map[string]LoadSplitter),
}

type testParamsRaw struct {
	Name           string
	Description    string
	TestType       string `json:"testType" yaml:"testType"`
	Params         json.RawMessage
	AttackDuration string `json:"attackDuration" yaml:"attackDuration"`
	NumMessages    int    `json:"numMessages" yaml:"numMessages"`
	Per            string
}

func (t TestParams) intoRaw() testParamsRaw {
	return testParamsRaw{
		AttackDuration: t.AttackDuration.String(),
		NumMessages:    t.NumMessages,
		TestType:       t.TestType,
		Per:            t.Per.String(),
		Name:           t.Name,
		Description:    t.Description,
		Params:         t.Params,
	}
}

func (raw testParamsRaw) into(result *TestParams) error {
	var err error

	if result == nil {
		return errors.New("into called with nil result")
	}

	var attackDuration time.Duration

	if len(raw.AttackDuration) > 0 {
		attackDuration, err = time.ParseDuration(raw.AttackDuration)
	}
	if err != nil {
		return fmt.Errorf("deserialization error, invalid duration %s passed to attackDuration", raw.AttackDuration)
	}

	var per time.Duration

	if len(raw.Per) > 0 {
		per, err = time.ParseDuration(raw.Per)
	}
	if err != nil {
		return fmt.Errorf("deserialization error, invalid duration %s passed to per", raw.Per)
	}
	result.AttackDuration = attackDuration
	result.Per = per
	result.TestType = raw.TestType
	result.NumMessages = raw.NumMessages
	result.Name = raw.Name
	result.Description = raw.Description
	result.Params = raw.Params
	return nil
}

func (t *TestParams) UnmarshalJSON(b []byte) error {
	if t == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw testParamsRaw
	var err error
	if err = json.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(t)
}

func (t TestParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.intoRaw())
}

func (t *TestParams) UnmarshalYaml(b []byte) error {
	if t == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw testParamsRaw

	var err error
	if err = yaml.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(t)
}

func (t TestParams) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(t.intoRaw())
}
