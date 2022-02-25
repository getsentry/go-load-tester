package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

func GetAuthHeader() string {
	//TODO need project key from settings (either CLI or config file)
	projectKey := "123"
	return fmt.Sprintf("Sentry sentry_key=%s,sentry_version=7", projectKey)
}

// SessionEnvelopeFromBody  creates the body of a session
// shamelessly stolen and modified from sentry-go/transport.go
func SessionEnvelopeFromBody(eventID string, sentAt time.Time, body json.RawMessage) (*bytes.Buffer, error) {

	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	// envelope header
	err := enc.Encode(struct {
		EventID string    `json:"event_id"`
		SentAt  time.Time `json:"sent_at"`
	}{
		EventID: eventID,
		SentAt:  sentAt,
	})
	if err != nil {
		return nil, err
	}
	// item header
	err = enc.Encode(struct {
		Type   string `json:"type"`
		Length int    `json:"length"`
	}{
		Type:   "session",
		Length: len(body),
	})
	if err != nil {
		return nil, err
	}
	// payload
	err = enc.Encode(body)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
