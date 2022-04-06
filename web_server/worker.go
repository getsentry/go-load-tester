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

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	vegeta "github.com/tsenart/vegeta/lib"

	"github.com/getsentry/go-load-tester/tests"
	"github.com/getsentry/go-load-tester/utils"
)

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
	//if working with master first wait to register
	config, err := registerWithMaster(port, masterUrl)
	if err != nil {
		log.Error().Err(err).Msg("Failed to register with master, worker stopping")
		return
	}
	go worker(targetUrl, statsdAddr, *config, paramChannel)
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
func createTargeter(targetUrl string, params tests.TestParams) vegeta.Targeter {
	log.Trace().Msgf("Creating targeter:%v", params)
	if params.AttackDuration == 0 {
		// an attack with 0 duration is a stop request
		log.Info().Msg("Stop command received")
		return nil
	}
	targeterBuilder := tests.GetTargeter(params.TestType)
	if targeterBuilder == nil {
		log.Error().Msgf("Invalid attack type %s", params.TestType)
		return nil
	}
	return targeterBuilder(targetUrl, params.Params)

}

// worker the worker that handles Vegeta attacks
//
// The worker uses a command channel to accept new commands
// Once a command is received the current attack (if there is a current attack)
// is stopped and a new attack started
func worker(targetUrl string, statsdAddr string, configParams configParams, paramsChan <-chan tests.TestParams) {

	if len(configParams.StatsdServerUrl) > 0 {
		//override configuration with master statsdUrl
		statsdAddr = configParams.StatsdServerUrl
	}
	if len(configParams.TargetUrl) > 0 {
		//override configuration with master targetUrl
		targetUrl = configParams.TargetUrl
	}
	log.Info().Msgf("Worker started targetUrl=%s, statsdAddr=%s", targetUrl, statsdAddr)
	var targeter vegeta.Targeter
	var params tests.TestParams
	var statsdClient = utils.GetStatsd(statsdAddr)
	for {
	attack:
		select {
		case params = <-paramsChan:
			targeter = createTargeter(targetUrl, params)
		default:
			if targeter != nil {
				rate := vegeta.Rate{Freq: params.NumMessages, Per: params.Per}
				attacker := vegeta.NewAttacker(vegeta.Timeout(time.Millisecond*500), vegeta.Redirects(0))
				for res := range attacker.Attack(targeter, rate, params.AttackDuration, params.Description) {
					if statsdClient != nil {
						var httpStatus = fmt.Sprintf("status:%d", res.Code)
						_ = statsdClient.Timing("req-latency", res.Latency, []string{httpStatus}, 1)
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
