package web_server

/*
Contains code for the Worker web server
*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"

	"github.com/getsentry/go-load-tester/tests"
	"github.com/getsentry/go-load-tester/utils"
)

var globalWorkerMetrics struct {
	vegetaStats vegeta.Metrics
}

// collectWorkerMetricsLoop regularly produces global master metrics
func collectWorkerMetricsLoop(statsdClient *statsd.Client) {
	if statsdClient == nil {
		return
	}

	tags := []string{}
	const sampleRate = 1.0
	const flushPeriod = 1 * time.Second
	const success_rate_threshold = 0.9
	const invalid_data_alert_threshold = 5

	// This counter will be increased if the success rate on the given step is lower
	// than `success_rate_threshold`
	invalid_data_counter := 0

	var lastFlushVegetaStats vegeta.Metrics

	for {
		invalid_data_marker := 0

		// Note: this is a shallow copy, but it should be fine if we don't access any thread-unsafe
		// attributes like maps/slices.
		currentVegetaStats := globalWorkerMetrics.vegetaStats
		currentVegetaStats.Histogram = nil
		currentVegetaStats.Latencies = vegeta.LatencyMetrics{}
		currentVegetaStats.Errors = make([]string, 0)
		currentVegetaStats.StatusCodes = make(map[string]int)
		currentVegetaStats.Close()
		log.Trace().Msgf("Current stats: %+v", currentVegetaStats)

		// Check if vegeta is struggling to reach the desired attack rate.
		// This might mean that the target is not ready to accept the desired traffic.
		// Only do the calculations if there's some data present.
		if !currentVegetaStats.Earliest.IsZero() && currentVegetaStats.Earliest == lastFlushVegetaStats.Earliest {
			requestsMade := currentVegetaStats.Requests - lastFlushVegetaStats.Requests
			successfulRequests := currentVegetaStats.Success*float64(currentVegetaStats.Requests) - lastFlushVegetaStats.Success*float64(lastFlushVegetaStats.Requests)
			successRate := 0.0
			if requestsMade > 0 {
				successRate = successfulRequests / float64(requestsMade)
			}
			log.Debug().Msgf("Over the last flush period, requests made: %d, successful requests: %.2f, success rate: %.2f", requestsMade, successfulRequests, successRate)

			if successRate < success_rate_threshold {
				invalid_data_counter += 1
			} else {
				invalid_data_counter = 0
			}

			if invalid_data_counter > invalid_data_alert_threshold {
				// The running test is most likely invalid
				invalid_data_marker = 1
			}
		}

		_ = statsdClient.Gauge("vegeta.data_invalid", float64(invalid_data_marker), tags, sampleRate)
		_ = statsdClient.Gauge("vegeta.rate", currentVegetaStats.Rate, tags, sampleRate)
		_ = statsdClient.Gauge("vegeta.throughput", currentVegetaStats.Throughput, tags, sampleRate)
		_ = statsdClient.Gauge("vegeta.success_pct", currentVegetaStats.Success, tags, sampleRate)
		_ = statsdClient.Gauge("vegeta.requests", float64(currentVegetaStats.Requests), tags, sampleRate)

		lastFlushVegetaStats = currentVegetaStats

		time.Sleep(flushPeriod)
	}
}

func RunWorkerWebServer(port string, targetUrl string, masterUrl string, statsdAddr string) {

	paramChannel := make(chan tests.TestParams)
	defer close(paramChannel)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	var statsdClient = utils.GetStatsd(statsdAddr)

	go collectWorkerMetricsLoop(statsdClient)

	engine.GET("/stop/", withParamChannel(paramChannel, workerStopHandler))
	engine.POST("/stop/", withParamChannel(paramChannel, workerStopHandler))
	engine.POST("/command/", withParamChannel(paramChannel, workerCommandHandler))
	engine.GET("/ping", pingHandler)
	engine.POST("/ping", pingHandler)
	//if working with master first wait to register
	config, err := registerWithMaster(port, masterUrl)
	if err != nil {
		log.Error().Err(err).Msg("Failed to register with master, worker stopping")
		return
	}
	go worker(targetUrl, statsdAddr, config, paramChannel)
	if len(port) > 0 {
		port = fmt.Sprintf(":%s", port)
	}
	_ = engine.SetTrustedProxies([]string{})
	_ = engine.Run(port)
}

type handlerWithCommand func(chan<- tests.TestParams, *gin.Context)

// registerWithMaster tries to register the current worker with a master
func registerWithMaster(port string, masterUrl string) (*configParams, error) {
	if len(masterUrl) == 0 {
		log.Info().Msg("No master url specified, running in independent mode")
		return nil, nil // do not try to register to master
	}
	registrationUrl := fmt.Sprintf("%s/register/", masterUrl)
	log.Info().Msgf("Trying to register with master at: %s", registrationUrl)
	c := http.Client{Timeout: time.Duration(2) * time.Second}

	ipAddr, err := utils.GetExternalIPv4()
	if err != nil {
		log.Err(err)
		return nil, err
	}
	workerUrl := fmt.Sprintf("http://%s:%s", ipAddr, port)
	body, err := createRegistrationBody(workerUrl)
	if err != nil {
		log.Error().Msgf("could not create registration body:\n%s", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", registrationUrl, body)
	if err != nil {
		log.Error().Msgf("Could not create registration request:\n%s", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	backoff := utils.ExponentialBackoff(time.Second*5, time.Second*30, 1.4)
	for {
		// try registering until success or unrecoverable error
		resp, err := c.Do(req)
		var status int
		var responseBody []byte
		if resp != nil {
			status = resp.StatusCode
			responseBody, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Error().Err(err).Msg("could not read response body")
			}
			err = resp.Body.Close()
			if err != nil {
				log.Error().Err(err).Msg("could not close response body")
			}
		}
		if err == nil {
			if status < 300 {
				// registration successful, unmarshal response
				log.Info().Msgf("Registration successful")
				var resp registerWorkerResponse
				err = json.Unmarshal(responseBody, &resp)
				if err != nil {
					log.Error().Err(err).Msg("could not deserialize master response")
					return nil, err
				}
				return &resp.Params, nil
			}
			if status >= 300 && status < 500 {
				err = fmt.Errorf("master returned: %d", status)
				// we can't handle redirects or client errors, no point in trying again
				log.Error().Err(err).Msgf("Client error returned from master")
				return nil, err
			}
		}
		nextTry := backoff()
		log.Error().Msgf("Failed to register with master trying again in %v, status:%d, err:%s\n", nextTry, status, err)
		// if we are here there was either a 5xx or some network error, try latter after backoff
		time.Sleep(nextTry)
	}
}

// createRegistrationBody creates the body of a "registration with master" request
//
// this is a JSON like e.g. {"workerUrl": "140.10.10.200:8088"}
func createRegistrationBody(workerUrl string) (*bytes.Buffer, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	err := enc.Encode(registerWorkerRequest{workerUrl})
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// withParamChannel constructs a Gin handler from a handler that also accepts a command channel
func withParamChannel(paramsChannel chan<- tests.TestParams, handler handlerWithCommand) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(paramsChannel, ctx)
	}
}

func pingHandler(ctx *gin.Context) {
	//just reply with a 200
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// workerStopHandler handle stop requests
func workerStopHandler(params chan<- tests.TestParams, ctx *gin.Context) {
	params <- tests.TestParams{} // send a "0" params will be interpreted as a "Stop request"
	ctx.String(http.StatusOK, "Stopping requested")
}

// workerCommandHandler handle command requests
func workerCommandHandler(cmd chan<- tests.TestParams, ctx *gin.Context) {
	var params tests.TestParams
	err := ctx.BindJSON(&params)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Could not parse body")
	}
	cmd <- params
	ctx.String(http.StatusOK, "Command Accepted")

}

// createTargeter creates a targeter for the passed test parameters
func createTargeter(targetUrl string, params tests.TestParams) tests.LoadTester {
	log.Trace().Msgf("Creating targeter:%v", params)
	if params.AttackDuration == 0 {
		// an attack with 0 duration is a stop request
		log.Info().Msg("Stop command received")
		return nil
	}
	loadTesterBuilder := tests.GetLoadTester(params.TestType)
	if loadTesterBuilder == nil {
		log.Error().Msgf("Invalid attack type %s", params.TestType)
		return nil
	}
	return loadTesterBuilder(targetUrl, params.Params)

}

// worker that handles Vegeta attacks
//
// The worker uses a command channel to accept new commands
// Once a command is received the current attack (if there is a current attack)
// is stopped and a new attack started
func worker(targetUrl string, statsdAddr string, configParams *configParams, paramsChan <-chan tests.TestParams) {

	if configParams != nil && len(configParams.StatsdServerUrl) > 0 {
		//override configuration with master statsdUrl
		statsdAddr = configParams.StatsdServerUrl
	}
	if configParams != nil && len(configParams.TargetUrl) > 0 {
		//override configuration with master targetUrl
		targetUrl = configParams.TargetUrl
	}
	log.Info().Msgf("Worker started targetUrl=%s, statsdAddr=%s", targetUrl, statsdAddr)
	var loadTester tests.LoadTester
	var params tests.TestParams
	var statsdClient = utils.GetStatsd(statsdAddr)
	globalWorkerMetrics.vegetaStats = vegeta.Metrics{}
	for {
	attack:
		select {
		case params = <-paramsChan:
			loadTester = createTargeter(targetUrl, params)
		default:
			if loadTester != nil {
				rate := vegeta.Rate{Freq: params.NumMessages, Per: params.Per}
				attacker := vegeta.NewAttacker(vegeta.Timeout(time.Millisecond*500), vegeta.Redirects(0), vegeta.MaxWorkers(1000))
				for res := range attacker.Attack(loadTester.GetTargeter(), rate, params.AttackDuration, params.Description) {
					globalWorkerMetrics.vegetaStats.Add(res)
					loadTester.ProcessResult(res)
					if statsdClient != nil {
						var httpStatus = fmt.Sprintf("status:%d", res.Code)
						_ = statsdClient.Timing("req-latency", res.Latency, []string{httpStatus}, 1.0)
					}
					select {
					case params = <-paramsChan:
						loadTester = createTargeter(targetUrl, params)
						attacker.Stop()

						// Flush stats
						globalWorkerMetrics.vegetaStats.Close()
						log.Debug().Msgf("Vegeta stats: %+v", globalWorkerMetrics.vegetaStats)
						globalWorkerMetrics.vegetaStats = vegeta.Metrics{}

						break attack // starts a new attack
					default:
						continue
					}
				}
				// finish current attack, reset timing
				loadTester = nil

				// Flush stats
				globalWorkerMetrics.vegetaStats.Close()
				log.Debug().Msgf("Vegeta stats: %+v", globalWorkerMetrics.vegetaStats)
				globalWorkerMetrics.vegetaStats = vegeta.Metrics{}
			} else {
				time.Sleep(1 * time.Second) // sleep a bit, so we don't busy spin when there is no attack
			}
		}
	}
}
