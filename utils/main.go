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

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func GetAuthHeader(projectKey string) string {
	//TODO need project key from settings (either CLI or config file)
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
func RandomChoice(choices []string, relativeWeights []int64) (string, error) {
	lc := len(choices)

	lr := len(relativeWeights)
	if lc == 0 {
		return "", errors.New("no valid choices")
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

	//cleanup relative weights
	for idx := 0; idx < len(relativeWeights); idx++ {
		if relativeWeights[idx] < 0 {
			relativeWeights[idx] = 0
		}
	}

	var maxWeight int64 = 0
	for _, weight := range relativeWeights {
		maxWeight += weight
	}
	var choice int64 = 0
	if maxWeight > 0 {
		choice = rand.Int63n(maxWeight)
	} else {
		return "", errors.New("no valid weights")
	}
	var curWeight int64 = 0
	for idx, weight := range relativeWeights {
		curWeight += weight
		if curWeight > choice {
			return choices[idx], nil
		}
	}
	// we shouldn't get here
	return "", errors.New("internal error RandomChoice")
}

// SimpleRandomChoice returns one of the given choices picked up randomly, with the same probability for each choice.
func SimpleRandomChoice(choices []string) string {
	weights := make([]int64, len(choices))
	for i := 0; i < len(weights); i++ {
		weights[i] = 1
	}
	retVal, _ := RandomChoice(choices, weights)
	return retVal
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
	return "", errors.New("no address found, make sure you are connected to the network")
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

// PerSecond converts a number of elements per random duration in elements per second
func PerSecond(elements int64, interval time.Duration) (float64, error) {
	if interval == 0 {
		return 0, errors.New("invalid 0 duration")
	}
	return float64(elements) * float64(time.Second) / float64(interval), nil
}

func GetStatsd(statsdAddr string) *statsd.Client {
	if len(statsdAddr) == 0 {
		log.Warn().Msgf("No statsd configured, will not emit stasd metrics")
		return nil
	}
	var client *statsd.Client
	//TODO find a better way to identify the current running worker (some Kubernetis magic ? )
	ip, err := GetExternalIPv4()
	if err != nil {
		log.Error().Err(err).Msg("Could not get worker IP, messages will not be tagged\n")
		client, err = statsd.New(statsdAddr)
	} else {
		var serverTag = fmt.Sprintf("ip:%s", ip)
		tagsOption := statsd.WithTags([]string{serverTag})
		client, err = statsd.New(statsdAddr, tagsOption)
	}
	if err != nil {
		log.Error().Err(err).Msgf("Could not connect to stastd backend")
		return nil
	}
	log.Info().Msgf("Registered with statsd server at: %s", statsdAddr)
	return client
}
