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

// ConfigurableTargeterBuilder is a function that when given a URL and a read channel of
// raw JSON messages returns a vegeta.Targeter that is able to change the events it generates
// to reflect the parameter passed through the JSON messages channel.
// Note: the raw JSON messages received through the channel need to be "compatible" with the specific
// targeter. Getting the proper builder for a type of message is outside this function's responsibilities
// (the dispatch is done via GetTargeter inside the worker)
type ConfigurableTargeterBuilder func(url string, params json.RawMessage) vegeta.Targeter

func RegisterTargeter(name string, builder ConfigurableTargeterBuilder) {
	converters.lock.Lock()
	defer converters.lock.Unlock()
	converters.targeterBuilders[name] = builder
}

// GetTargeter returns the TargeterBuilder for a particular type of message. The name represents the
// type of load message passed
func GetTargeter(testType string) ConfigurableTargeterBuilder {
	converters.lock.Lock()
	defer converters.lock.Unlock()
	return converters.targeterBuilders[testType]
}

var converters = struct {
	targeterBuilders map[string]ConfigurableTargeterBuilder
	lock             sync.Mutex
}{
	targeterBuilders: make(map[string]ConfigurableTargeterBuilder),
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
