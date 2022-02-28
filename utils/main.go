package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"strings"
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

// RandomChoice implements the weighted version of the python random.choices standard function
//
// if relativeWeights is empty or smaller than choices weights of 1 will be considered for the
// missing weights, if more weights are passed they will be ignored
func RandomChoice(choices []string, relativeWeights []int64) string {
	lc := len(choices)
	lr := len(relativeWeights)
	if lc == 0 {
		return ""
	}
	if lr > lc {
		relativeWeights = relativeWeights[:lc]
	}
	if lc > lr {
		x := make([]int64, lc-lr, lc-lr)
		for i := range x {
			x[i] = 1
		}
		relativeWeights = append(relativeWeights, x...)
	}

	var maxWeight int64 = 0
	for _, weight := range relativeWeights {
		maxWeight += weight
	}
	choice := rand.Int63n(maxWeight)
	var curWeight int64 = 0
	for idx, weight := range relativeWeights {
		curWeight += weight
		if curWeight > choice {
			return choices[idx]
		}
	}
	// we shouldn't get here
	log.Printf("Internal error RandomChoice")
	return choices[lc-1]

}

// UuidAsHex similar with uuid.hex from python ( returns the UUID as a hex string without - )
func UuidAsHex(id uuid.UUID) string {
	idStr := id.String()
	return strings.Replace(idStr, "-", "", -1)
}
