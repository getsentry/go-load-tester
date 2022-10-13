package utils

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

// StringDuration a duration that serializes in Json/Yaml as a string
// e.g. "2m"
type StringDuration time.Duration

func (t *StringDuration) UnmarshalJSON(b []byte) error {
	var buffer string

	if t == nil {
		return errors.New("nil pointer passed to UnmarshalJSON")
	}

	err := json.Unmarshal(b, &buffer)

	if err != nil {
		log.Error().Err(err).Msg("Failed to deserialize buffer into a string")
		return err
	}
	return t.fromString(buffer)
}

func (t StringDuration) MarshalJSON() ([]byte, error) {
	var d time.Duration = time.Duration(t)
	return json.Marshal(d.String())
}

func (t *StringDuration) UnmarshalYAML(unmarshal func(any) error) error {
	var buffer string

	if t == nil {
		return errors.New("nil pointer passed to UnmarshalYaml")
	}

	err := unmarshal(&buffer)

	if err != nil {
		log.Error().Err(err).Msg("Failed to deserialize buffer into a string")
		return err
	}
	return t.fromString(buffer)
}

func (t StringDuration) MarshalYAML() (any, error) {
	var d time.Duration = time.Duration(t)
	return d.String(), nil
}

func (t *StringDuration) fromString(s string) error {
	var duration time.Duration
	var err error

	duration, err = time.ParseDuration(s)

	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse %s  into a Duration", s)
		return err
	}
	*t = StringDuration(duration)
	return nil
}
