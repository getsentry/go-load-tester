package tests

// Contains  functionality for generating Session load tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"
	"gopkg.in/yaml.v2"

	"github.com/getsentry/go-load-tester/utils"
)

const timeFormat = "2006-01-02T03:04:05.000Z"

// SessionJob is how a session load test is parameterized
type SessionJob struct {
	StartedRange    time.Duration
	DurationRange   time.Duration
	NumReleases     int64
	NumEnvironments int64
	NumUsers        int64
	OkWeight        int64
	ExitedWeight    int64
	ErroredWeight   int64
	CrashedWeight   int64
	AbnormalWeight  int64
}

// Session serialisation format for sessions
type Session struct {
	Init       bool    `json:"init"`
	Started    string  `json:"started"` // a date time
	Status     string  `json:"status"`  // ok, exited, errored, crashed, abnormal
	Errors     int64   `json:"errors"`
	Duration   float64 `json:"duration"` // duration in seconds
	SessionId  string  `json:"sid"`
	UserId     string  `json:"did,omitempty"`
	Sequence   int64   `json:"seq"`
	Timestamp  string  `json:"timestamp"` // a date time
	Attributes struct {
		Release     string `json:"release"`
		Environment string `json:"environment"`
	} `json:"attrs"`
}

// NewSessionTargeter returns a targeter that listens over a channel for changes
// to the session generation spec and creates requests (i.e. Target(s)) matching
// the configuration
func NewSessionTargeter(url string, rawSession json.RawMessage) vegeta.Targeter {
	var sessionParams SessionJob
	err := json.Unmarshal(rawSession, &sessionParams)
	if err != nil {
		log.Error().Err(err).Msgf("invalid session params received\nraw data\n%s",
			rawSession)
	}

	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		tgt.Method = "POST"

		//TODO add more sophisticated projectId/projectKey generation
		projectId := "1"
		projectKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1"

		tgt.URL = fmt.Sprintf("%s/api/%s/envelope/", url, projectId)
		tgt.Header = make(http.Header)
		tgt.Header.Set("X-Sentry-Auth", utils.GetAuthHeader(projectKey))
		tgt.Header.Set("Content-Type", "application/x-sentry-envelope")

		body, err := getSessionBody(sessionParams)
		if err != nil {
			return err
		}

		tgt.Body = body
		log.Trace().Msg("Attacking")
		return nil
	}
}

func getSessionBody(sp SessionJob) ([]byte, error) {
	var session Session
	log.Trace().Msgf("session job: %v", sp)

	// Logic copied from ingest-load-tester session_event_task_factory
	maxDurationDeviation := sp.DurationRange
	maxStartDeviation := sp.StartedRange
	now := time.Now().UTC()
	baseStart := now.Add(-maxStartDeviation - maxDurationDeviation)
	if maxDurationDeviation < time.Millisecond {
		maxDurationDeviation = time.Millisecond
	}
	startDeviation := time.Duration(rand.Int63n(int64(maxDurationDeviation)))
	staredTime := baseStart.Add(startDeviation)
	if maxStartDeviation < time.Second {
		maxStartDeviation = time.Millisecond
	}
	duration := float64(rand.Int63n(int64(maxStartDeviation))) / float64(time.Second)
	started := staredTime.Format(timeFormat)
	timestamp := now.Format(timeFormat)
	release := fmt.Sprintf("r-1.0.%d", rand.Int63n(sp.NumReleases))
	environment := fmt.Sprintf("environment-%d", rand.Int63n(sp.NumEnvironments))
	status, err := utils.RandomChoice([]string{"ok", "exited", "errored", "crashed", "abnormal"},
		[]int64{sp.OkWeight, sp.ExitedWeight, sp.ErroredWeight, sp.CrashedWeight, sp.AbnormalWeight})
	if err != nil {
		status = "ok"
	}
	init := true
	seq := int64(0)

	if status != "ok" {
		init = false
		seq = rand.Int63n(5)
	}

	var errs int64 = 0
	if status == "errored" {
		errs = rand.Int63n(19) + 1
	}

	userId := fmt.Sprintf("u-%d", rand.Int63n(sp.NumUsers))
	sessionId, err := uuid.NewUUID()
	sessionIdStr := utils.UuidAsHex(sessionId)
	eventId, err := uuid.NewUUID()
	eventIdStr := utils.UuidAsHex(eventId)

	session = Session{
		Init:      init,
		Started:   started,
		Status:    status,
		Errors:    errs,
		Duration:  duration,
		SessionId: sessionIdStr,
		UserId:    userId,
		Timestamp: timestamp,
		Sequence:  seq,
	}
	session.Attributes.Environment = environment
	session.Attributes.Release = release

	body, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	var buff *bytes.Buffer

	buff, err = utils.EnvelopeFromBody(eventIdStr, now, "session", map[string]string{}, body)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil

}

//TODO Check if there is a cleaner way to do serialisation.

func (s *SessionJob) UnmarshalJSON(b []byte) error {
	if s == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw sessionJobRaw
	var err error
	if err = json.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(s)
}

func (s SessionJob) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.intoRaw())
}

func (s *SessionJob) UnmarshalYaml(b []byte) error {
	if s == nil {
		return errors.New("nil value passed as deserialization target")
	}
	var raw sessionJobRaw

	var err error
	if err = yaml.Unmarshal(b, &raw); err != nil {
		return err
	}
	return raw.into(s)
}

func (s SessionJob) MarshalYaml() ([]byte, error) {
	return yaml.Marshal(s.intoRaw())
}

func (s SessionJob) intoRaw() sessionJobRaw {
	return sessionJobRaw{
		StartedRange:    s.StartedRange.String(),
		DurationRange:   s.DurationRange.String(),
		NumReleases:     s.NumReleases,
		NumEnvironments: s.NumEnvironments,
		NumUsers:        s.NumUsers,
		OkWeight:        s.OkWeight,
		ExitedWeight:    s.ExitedWeight,
		CrashedWeight:   s.CrashedWeight,
		AbnormalWeight:  s.AbnormalWeight,
		ErroredWeight:   s.ErroredWeight,
	}
}

func (raw sessionJobRaw) into(result *SessionJob) error {
	var err error

	if result == nil {
		return errors.New("into called with nil result")
	}

	var startedRange time.Duration

	if len(raw.StartedRange) > 0 {
		startedRange, err = time.ParseDuration(raw.StartedRange)
	}
	if err != nil {
		return fmt.Errorf("deserialization error, invalid duration %s passed to startedRange", raw.StartedRange)
	}

	var durationRange time.Duration

	if len(raw.DurationRange) > 0 {
		durationRange, err = time.ParseDuration(raw.DurationRange)
	}
	if err != nil {
		return fmt.Errorf("deserialization error, invalid duration %s passed to durationRange", raw.DurationRange)
	}
	result.StartedRange = startedRange
	result.DurationRange = durationRange
	result.NumReleases = raw.NumReleases
	result.NumEnvironments = raw.NumEnvironments
	result.NumUsers = raw.NumUsers
	result.OkWeight = raw.OkWeight
	result.ExitedWeight = raw.ExitedWeight
	result.ErroredWeight = raw.ErroredWeight
	result.CrashedWeight = raw.CrashedWeight
	result.AbnormalWeight = raw.AbnormalWeight
	return nil
}

/// Struct used for serialisation
type sessionJobRaw struct {
	StartedRange    string `json:"startedRange" yaml:"startedRange"`
	DurationRange   string `json:"durationRange" yaml:"durationRange"`
	NumReleases     int64  `json:"numReleases" yaml:"numReleases"`
	NumEnvironments int64  `json:"numEnvironments" yaml:"numEnvironments"`
	NumUsers        int64  `json:"numUsers" yaml:"numUsers"`
	OkWeight        int64  `json:"okWeight" yaml:"okWeight"`
	ExitedWeight    int64  `json:"exitedWeight" yaml:"exitedWeight"`
	ErroredWeight   int64  `json:"erroredWeight" yaml:"erroredWeight"`
	CrashedWeight   int64  `json:"crashedWeight" yaml:"crashedWeight"`
	AbnormalWeight  int64  `json:"abnormalWeight" yaml:"abnormalWeight"`
}

func init() {
	RegisterTargeter("session", NewSessionTargeter)
}
