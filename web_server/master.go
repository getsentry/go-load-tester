package web_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/getsentry/go-load-tester/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/getsentry/go-load-tester/tests"
)

/*
Contains code for the Master web server
*/

var masterState struct {
	lock    sync.Mutex
	workers []string // urls of registered workers
}

// getWorkers returns a copy of the workers at the moment of calling
//
// Use it to safely get a copy and then release the lock on the masterState
func getWorkers() []string {
	masterState.lock.Lock()
	defer masterState.lock.Unlock()
	var retVal = make([]string, len(masterState.workers))
	copy(retVal, masterState.workers)
	return retVal
}

func okJsonResponse() interface{} {
	return gin.H{"status": "ok"}
}

func errorJsonResponse(errorMessage string) interface{} {
	return gin.H{"error": errorMessage}
}

// getDefaultHttpClient returns a correctly configured HTTP Client for passing
// requests to workers (a common point to configure options for worker requests)
func getDefaultHttpClient() http.Client {
	return http.Client{Timeout: time.Duration(1) * time.Second}
}

func addWorker(url string) {
	masterState.lock.Lock()
	defer masterState.lock.Unlock()
	for _, workerUrl := range masterState.workers {
		if workerUrl == url {
			// worker already registered
			return
		}
	}
	masterState.workers = append(masterState.workers, url)
	log.Info().Msgf("Register worker at: %s", url)
}

func removeWorker(url string) {
	masterState.lock.Lock()
	defer masterState.lock.Unlock()
	var l = len(masterState.workers)

	for idx, workerUrl := range masterState.workers {
		if workerUrl == url {
			masterState.workers[idx] = masterState.workers[l-1]
			masterState.workers = masterState.workers[:l-1]
			return
		}
	}
}

func RunMasterWebServer(port string, statsdAddr string) {
	gin.SetMode(gin.ReleaseMode)
	var engine = gin.Default()
	var statsd = utils.GetStatsd(statsdAddr)

	engine.GET("/stop/", masterStopHandler)
	engine.POST("/stop/", masterStopHandler)
	engine.POST("/command/", handlerWithStatsd(statsd, masterCommandHandler))
	engine.POST("/register/", masterRegisterHandler)
	engine.POST("/unregister/", masterUnregisterHandler)
	if len(port) > 0 {
		port = fmt.Sprintf(":%s", port)
	}
	_ = engine.SetTrustedProxies([]string{})
	_ = engine.Run(port)
}

func handlerWithStatsd(statsdClient *statsd.Client, handler func(*statsd.Client, *gin.Context)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(statsdClient, ctx)
	}
}

func ForwardAttack(params tests.TestParams) {
	checkWorkersStatus()
	var workerUrls = getWorkers()
	if len(workerUrls) == 0 {
		log.Error().Msg("Cannot forward attack, no workers registered")
		return
	}
	// divide attack intensity among workers
	params.Per = time.Duration(len(workerUrls)) * params.Per
	newParams, err := json.Marshal(params)
	if err != nil {
		log.Error().Msgf("Error generating request%s", err)
		return
	}

	client := getDefaultHttpClient()

	for _, workerUrl := range workerUrls {
		go func(workerUrl string) {
			body := bytes.NewReader(newParams)
			var commandUrl = fmt.Sprintf("%s/command/", workerUrl)
			req, err := http.NewRequest("POST", commandUrl, body)
			if err != nil {
				log.Err(err)
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				log.Error().Msgf(" error sending command to client '%s': \n%s", workerUrl, err)
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
		}(workerUrl)
	}
}

// checkWorkersStatus checks all clients ping endpoint to verify that they are still working
func checkWorkersStatus() {
	var workerUrls = getWorkers()

	if len(workerUrls) == 0 {
		return
	}
	var waitClientPings sync.WaitGroup
	var client = getDefaultHttpClient()
	waitClientPings.Add(len(workerUrls))
	for _, worker := range workerUrls {
		go func(workerUrl string) {
			defer waitClientPings.Done()
			var pingUrl = fmt.Sprintf("%s/ping/", workerUrl)
			var resp, err = client.Get(pingUrl)
			defer func() {
				if resp != nil {
					_ = resp.Body.Close()
				}
			}()
			if err != nil || (resp != nil && resp.StatusCode > 300) {
				removeWorker(workerUrl)
			}

		}(worker)
	}
	waitClientPings.Wait()
}

func masterStopHandler(ctx *gin.Context) {
	//no need to refresh clients
	log.Info().Msg("stop handler called")
	var workerUrls = getWorkers()
	var client = getDefaultHttpClient()
	for _, worker := range workerUrls {
		go func(workerUrl string) {
			var pingUrl = fmt.Sprintf("%s/stop/", workerUrl)
			var resp, err = client.Get(pingUrl)
			if err != nil {
				log.Error().Msgf("Could not send request to client %s", workerUrl)
			}
			defer resp.Body.Close()

		}(worker)
	}
	ctx.JSON(http.StatusOK, okJsonResponse())
}

func masterCommandHandler(statsdClient *statsd.Client, ctx *gin.Context) {
	var params tests.TestParams
	log.Info().Msg("command handler called")
	if err := ctx.ShouldBindJSON(&params); err != nil {
		log.Error().Msg("Could not parse command")
		ctx.JSON(http.StatusBadRequest, "Could not parse command")
		return
	}
	freq, err := utils.PerSecond(int64(params.NumMessages), params.Per)
	if err != nil {
		log.Error().Msgf("Failed to calculate request frequency for %d per %v", params.NumMessages, params.Per)
		return
	}
	if statsdClient != nil {
		statsdClient.Gauge("desired-req-sec", freq, []string{}, 1.0)
	}
	go ForwardAttack(params) // no need to wait for sending it to clients
	ctx.JSON(http.StatusOK, "Attack forwarded to workers")

}

func masterRegisterHandler(ctx *gin.Context) {
	var workerReq registerWorkerRequest
	if err := ctx.ShouldBindJSON(&workerReq); err == nil {
		addWorker(workerReq.WorkerUrl)
		ctx.JSON(http.StatusOK, okJsonResponse())
	} else {
		log.Error().Msg("Error while trying to register worker")
		ctx.JSON(http.StatusBadRequest, errorJsonResponse("Could not parse registration request"))
	}
}

func masterUnregisterHandler(ctx *gin.Context) {
	var workerReq registerWorkerRequest
	if err := ctx.ShouldBindJSON(&workerReq); err == nil {
		removeWorker(workerReq.WorkerUrl)
		ctx.JSON(http.StatusOK, okJsonResponse())
	} else {
		log.Error().Msg("Error while trying to unregister worker")
		ctx.JSON(http.StatusBadRequest, errorJsonResponse("Could not register request"))
	}
}
