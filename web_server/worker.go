package web_server

/*
Contains code for the Worker web server
*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"

	"github.com/getsentry/go-load-tester/tests"
	"github.com/getsentry/go-load-tester/utils"
)

// registerWorkerRequest is the body of the register http request sent by a worker
// to register to a master.
type registerWorkerRequest struct {
	WorkerUrl string `json:"workerUrl"`
}

func RunWorkerWebServer(port string, targetUrl string, masterUrl string, statsdAddr string) {

	paramChannel := make(chan tests.TestParams)
	defer close(paramChannel)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/stop/", withParamChannel(paramChannel, workerStopHandler))
	engine.POST("/stop/", withParamChannel(paramChannel, workerStopHandler))
	engine.POST("/command/", withParamChannel(paramChannel, workerCommandHandler))
	engine.GET("/ping", pingHandler)
	engine.POST("/ping", pingHandler)
	go worker(targetUrl, statsdAddr, paramChannel)
	go registerWithMaster(port, masterUrl)
	if len(port) > 0 {
		port = fmt.Sprintf(":%s", port)
	}
	_ = engine.SetTrustedProxies([]string{})
	_ = engine.Run(port)
}

type handlerWithCommand func(chan<- tests.TestParams, *gin.Context)

// registerWithMaster tries to register the current worker with a master
func registerWithMaster(port string, masterUrl string) {
	if len(masterUrl) == 0 {
		log.Info().Msg("No master url specified, running in independent mode")
		return // do not try to register to master
	}
	registrationUrl := fmt.Sprintf("%s/register/", masterUrl)
	log.Info().Msgf("Trying to register with master at: %s", registrationUrl)
	c := http.Client{Timeout: time.Duration(2) * time.Second}

	ipAddr, err := utils.GetExternalIPv4()
	if err != nil {
		log.Err(err)
		return
	}
	workerUrl := fmt.Sprintf("http://%s:%s", ipAddr, port)
	body, err := createRegistrationBody(workerUrl)
	if err != nil {
		log.Error().Msgf("could not create registration body:\n%s", err)
		return
	}
	req, err := http.NewRequest("POST", registrationUrl, body)
	if err != nil {
		log.Error().Msgf("Could not create registration request:\n%s", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	backoff := utils.ExponentialBackoff(time.Second*5, time.Second*30, 1.4)
	for {
		// try registering until success or unrecoverable error
		resp, err := c.Do(req)
		var status int
		if resp != nil {
			status = resp.StatusCode
			_ = resp.Body.Close()
		}
		if err == nil {
			if status < 300 {
				// registration successful
				log.Info().Msgf("Registration successful")
				break
			}
			if status >= 300 && status < 500 {
				// we can't handle redirects or client errors, no point in trying again
				log.Error().Msgf("error returned from master: %d", status)
				break
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
func createTargeter(targetUrl string, params tests.TestParams) vegeta.Targeter {
	if params.AttackDuration == 0 {
		log.Info().Msg("Zero attack duration, stopping")
		return nil
	}
	targeterBuilder := tests.GetTargeter(params.Name)
	if targeterBuilder == nil {
		log.Error().Msgf("Invalid attack type %s", params.Name)
		return nil
	}
	return targeterBuilder(targetUrl, params.Params)

}

// worker the worker that handles Vegeta attacks
//
// The worker uses a command channel to accept new commands
// Once a command is received the current attack (if there is a current attack)
// is stopped and a new attack started
func worker(targetUrl string, statsdAddr string, paramsChan <-chan tests.TestParams) {
	var targeter vegeta.Targeter
	var params tests.TestParams
	var statsdClient = getStatsd(statsdAddr)
	var noTags = []string{}
	for {
	attack:
		select {
		case params = <-paramsChan:
			targeter = createTargeter(targetUrl, params)
		default:
			if targeter != nil {
				// var metrics vegeta.Metrics // not used at the moment metrics.Add(res)
				rate := vegeta.Rate{Freq: params.NumMessages, Per: params.Per}
				attacker := vegeta.NewAttacker(vegeta.Timeout(time.Millisecond*500), vegeta.Redirects(0))
				for res := range attacker.Attack(targeter, rate, params.AttackDuration, params.Description) {
					fmt.Printf("%s", res.Latency)
					if statsdClient != nil {
						_ = statsdClient.Timing("req-latency", res.Latency, noTags, 1)
					}
					select {
					case params = <-paramsChan:
						targeter = createTargeter(targetUrl, params)
						attacker.Stop()
						break attack // starts a new attack
					default:
						continue
					}
				}
				// finish current attack reset timing
				targeter = nil
				time.Sleep(1 * time.Second) // sleep a bit, so we don't busy spin when there is no attack
			}
		}
	}
}

func getStatsd(statsdAddr string) *statsd.Client {
	if len(statsdAddr) == 0 {
		log.Warn().Msgf("No statsd configured, will not emit stasd metrics")
		return nil
	}
	var client, err = statsd.New(statsdAddr)
	if err != nil {
		log.Error().Msgf("Could not connect to stastd backend\n%v", err)
		return nil
	}
	//TODO find a better way to identify the current running worker (some Kubernetis magic ? )
	ip, err := utils.GetExternalIPv4()
	if err != nil {
		log.Error().Msgf("Could not get worker IP, messages will not be tagged\n%s", err)
	} else {
		var serverTag = fmt.Sprintf("worker_ip=%s", ip)
		statsd.WithTags([]string{serverTag})
	}
	log.Info().Msgf("Registered with statsd server at: %s", statsdAddr)
	return client
}
