package dataproviders

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// Basic interface that produces a value
type Value interface {
	GetValue(sequence uint64) interface{}
}

// This value always returns the same value
type ConstantValue struct {
	value interface{}
}

func NewConst(
	value interface{},
) *ConstantValue {
	return &ConstantValue{
		value: value,
	}
}

func NewConstFromConfig(config interface{}) (*ConstantValue, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	val, exists := value_config["value"]
	if !exists {
		return nil, fmt.Errorf("missing value attribute")
	}
	return NewConst(val), nil
}

func (constant *ConstantValue) GetValue(
	sequence uint64,
) interface{} {
	return constant.value
}

// Sequence of integers given a start and step.
type Sequence struct {
	From uint64
	Step uint64
}

func (seq *Sequence) GetValue(
	sequence uint64,
) interface{} {
	return seq.From + seq.Step*sequence
}

// Discrete values
type RandomSet struct {
	alphabet []string
}

func (seq *RandomSet) GetValue(
	sequence uint64,
) interface{} {
	randomIndex := rand.Intn(len(seq.alphabet))
	return seq.alphabet[randomIndex]
}

type SequenceSet struct {
	alphabet []string
}

func (seq *SequenceSet) GetValue(
	sequence uint64,
) interface{} {
	return seq.alphabet[sequence%uint64(len(seq.alphabet))]
}

func getAlphabet(config interface{}) ([]string, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	val, exists := value_config["alphabet"]
	if !exists {
		return nil, fmt.Errorf("missing value attribute")
	}

	var alphabet [](string)
	for _, word := range val.([](interface{})) {
		alphabet = append(alphabet, word.(string))
	}

	return alphabet, nil
}

func NewRandomSetFromConfig(config interface{}) (*RandomSet, error) {
	alphabet, err := getAlphabet((config))
	if err != nil {
		return nil, err
	}
	return &RandomSet{
		alphabet: alphabet,
	}, nil
}

func NewSequenceSetFromConfig(config interface{}) (*SequenceSet, error) {
	alphabet, err := getAlphabet((config))
	if err != nil {
		return nil, err
	}
	return &SequenceSet{
		alphabet: alphabet,
	}, nil
}

// Timestamp
type Timestamp struct {
	format string
}

func (seq *Timestamp) GetValue(
	sequence uint64,
) interface{} {
	now := time.Now()
	return now.Format(seq.format)
}

func NewTimestampFromConfig(config interface{}) (*Timestamp, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	val, exists := value_config["format"]
	if !exists {
		return nil, fmt.Errorf("missing value attribute")
	}
	return &Timestamp{
		format: val.(string),
	}, nil
}

// RandomTimestamp
type RandomTimestamp struct {
    start  time.Time
    end    time.Time
    format string
}

func (rt *RandomTimestamp) GetValue(sequence uint64,) interface{} {
    duration := rt.end.Sub(rt.start)
    randomDuration := time.Duration(rand.Int63n(int64(duration)))
    randomTime := rt.start.Add(randomDuration)
    return randomTime.Format(rt.format)
}

func NewRandomTimestampFromConfig(config interface{}) (*RandomTimestamp, error) {
    valueConfig, ok := config.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("config type invalid %s", valueConfig)
    }
    startVal, exists := valueConfig["start"]
    if !exists {
        return nil, fmt.Errorf("missing start attribute")
    }
    endVal, exists := valueConfig["end"]
    if !exists {
        return nil, fmt.Errorf("missing end attribute")
    }
    formatVal, exists := valueConfig["format"]
    if !exists {
        return nil, fmt.Errorf("missing format attribute")
    }

    format := formatVal.(string)
    start, err := time.Parse(format, startVal.(string))
    if err != nil {
        return nil, fmt.Errorf("invalid start time format: %v", err)
    }
    end, err := time.Parse(format, endVal.(string))
    if err != nil {
        return nil, fmt.Errorf("invalid end time format: %v", err)
    }

    return &RandomTimestamp{
        start:  start,
        end:    end,
        format: format,
    }, nil
}

// UUID
type UUIDGenerator struct{}

func (seq *UUIDGenerator) GetValue(
	sequence uint64,
) interface{} {
	return uuid.New()
}

// Random values

type RandomInteger struct {
	min int
	max int
}

func NewRandomIntegerFromConfig(config interface{}) (*RandomInteger, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	min, exists := value_config["min"]
	if !exists {
		return nil, fmt.Errorf("missing min attribute")
	}
	max, exists := value_config["max"]
	if !exists {
		return nil, fmt.Errorf("missing max attribute")
	}
	// TODO: Figure out why I cannot use integers here.
	return &RandomInteger{
		min: int(min.(float64)),
		max: int(max.(float64)),
	}, nil
}

func (rnd *RandomInteger) GetValue(
	sequence uint64,
) interface{} {
	if rnd.max == rnd.min {
		return rnd.min
	}
	randomIndex := rand.Intn(rnd.max - rnd.min)
	return rnd.min + randomIndex
}

type RandomFloat struct {
	min float64
	max float64
}

func NewRandomFloatFromConfig(config interface{}) (*RandomFloat, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	min, exists := value_config["min"]
	if !exists {
		return nil, fmt.Errorf("missing min attribute")
	}
	max, exists := value_config["max"]
	if !exists {
		return nil, fmt.Errorf("missing max attribute")
	}
	return &RandomFloat{
		min: min.(float64),
		max: max.(float64),
	}, nil
}

func (rnd *RandomFloat) GetValue(
	sequence uint64,
) interface{} {
	randomVal := rand.Float64() * (rnd.max - rnd.min)
	return rnd.min + randomVal
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomString(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type RandomString struct {
	minSize int
	maxSize int
}

func NewRandomStringFromConfig(config interface{}) (*RandomString, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	minSize, exists := value_config["minSize"]
	if !exists {
		return nil, fmt.Errorf("missing min attribute")
	}
	maxSize, exists := value_config["maxSize"]
	if !exists {
		return nil, fmt.Errorf("missing max attribute")
	}
	return &RandomString{
		minSize: int(minSize.(float64)),
		maxSize: int(maxSize.(float64)),
	}, nil
}

func (rnd *RandomString) GetValue(
	sequence uint64,
) interface{} {
	var length int
	if rnd.maxSize != rnd.minSize {

		length = rand.Intn(rnd.maxSize - rnd.minSize)
	} else {
		length = rnd.minSize
	}

	return generateRandomString(length)
}

// Array
type RandomArray struct {
	minSize       int
	maxSize       int
	valueProvider Value
}

func NewRandomArrayFromConfig(config interface{}, valueProvider Value) (*RandomArray, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	minSize, exists := value_config["minSize"]
	if !exists {
		return nil, fmt.Errorf("missing min attribute")
	}
	maxSize, exists := value_config["maxSize"]
	if !exists {
		return nil, fmt.Errorf("missing max attribute")
	}

	return &RandomArray{
		minSize:       int(minSize.(float64)),
		maxSize:       int(maxSize.(float64)),
		valueProvider: valueProvider,
	}, nil
}

func (rnd *RandomArray) GetValue(
	sequence uint64,
) interface{} {
	var length int
	if rnd.maxSize != rnd.minSize {

		length = rand.Intn(rnd.maxSize - rnd.minSize)
	} else {
		length = rnd.minSize
	}

	var ret [](interface{})
	for i := 0; i < length; i++ {
		new_val := rnd.valueProvider.GetValue(sequence)
		ret = append(ret, new_val)
	}
	return ret
}

// Map
type RandomMap struct {
	minSize       int
	maxSize       int
	keyProvider   Value
	valueProvider Value
}

func NewRandomMapFromConfig(config interface{}, keyProvider Value, valueProvider Value) (*RandomMap, error) {
	value_config, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config type invalid %s", value_config)
	}
	minSize, exists := value_config["minSize"]
	if !exists {
		return nil, fmt.Errorf("missing min attribute")
	}
	maxSize, exists := value_config["maxSize"]
	if !exists {
		return nil, fmt.Errorf("missing max attribute")
	}

	return &RandomMap{
		minSize:       int(minSize.(float64)),
		maxSize:       int(maxSize.(float64)),
		keyProvider:   keyProvider,
		valueProvider: valueProvider,
	}, nil
}

func (rnd *RandomMap) GetValue(
	sequence uint64,
) interface{} {
	var length int
	if rnd.maxSize != rnd.minSize {

		length = rand.Intn(rnd.maxSize - rnd.minSize)
	} else {
		length = rnd.minSize
	}

	var ret = make(map[string]interface{})
	for i := 0; i < length; i++ {
		new_key := rnd.keyProvider.GetValue(sequence)
		new_val := rnd.valueProvider.GetValue(sequence)

		ret[new_key.(string)] = new_val
	}
	return ret
}
