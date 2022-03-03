package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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
	log.Error().Msg("Internal error RandomChoice")
	return choices[lc-1]

}

// UuidAsHex similar with uuid.hex from python ( returns the UUID as a hex string without - )
func UuidAsHex(id uuid.UUID) string {
	idStr := id.String()
	return strings.Replace(idStr, "-", "", -1)
}

func GetExternalIPv4() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

// ExponentialBackoff returns an exponentially increasing Duration
//
// the duration will increase until the maximum duration is reached after which it will
// return that duration forever.
// This is not thread safe and should only be called from one goroutine per backoff function.
func ExponentialBackoff(initial time.Duration, maximum time.Duration, factor float64) func() time.Duration {
	if factor < 1 {
		log.Warn().Msgf("ExponentialBackoff called with invalid backoff factor %f, factor should be > 1, will set it to 2", factor)
		factor = 2
	}
	current := initial

	return func() time.Duration {
		retVal := current
		current = time.Duration(float64(current) * factor)

		if retVal > maximum {
			return maximum
		}
		return retVal
	}
}
