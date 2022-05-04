package web_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/getsentry/go-load-tester/tests"
	"github.com/getsentry/go-load-tester/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

/*
Contains code for the Master web server
*/

var masterState struct {
	lock    sync.Mutex
	workers []string // urls of registered workers
}

var globalMasterMetrics struct {
	desiredRate float64
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
	log.Info().Msgf("Registered worker at: %s", url)
}

func removeWorker(url string) {
	masterState.lock.Lock()
	defer masterState.lock.Unlock()
	var l = len(masterState.workers)

	for idx, workerUrl := range masterState.workers {
		if workerUrl == url {
			masterState.workers[idx] = masterState.workers[l-1]
			masterState.workers = masterState.workers[:l-1]
			log.Info().Msgf("Removed worker: %s", url)
			log.Debug().Msgf("Remaining workers: %v", masterState.workers)
			return
		}
	}
	log.Error().Msgf("Cannot remove worker: %v", url)
}

// collectMasterMetricsLoop regularly produces global master metrics
func collectMasterMetricsLoop(statsdClient *statsd.Client) {
	if statsdClient == nil {
		return
	}

	tags := []string{}
	sampleRate := 1.0
	flushPeriod := 1 * time.Second

	for {
		_ = statsdClient.Gauge("registered-workers", float64(len(masterState.workers)), tags, sampleRate)
		_ = statsdClient.Gauge("desired-req-sec", globalMasterMetrics.desiredRate, tags, sampleRate)

		time.Sleep(flushPeriod)
	}
}

func RunMasterWebServer(port string, statsdAddr string, targetUrl string) {
	gin.SetMode(gin.ReleaseMode)
	var engine = gin.Default()
	var statsdClient = utils.GetStatsd(statsdAddr)

	go collectMasterMetricsLoop(statsdClient)

	engine.Static("/static", "./static")
	engine.LoadHTMLGlob("templates/*.html")

	engine.GET("/docs", mainDocsHandler)
	engine.GET("/stop/", masterStopHandler)
	engine.POST("/stop/", masterStopHandler)
	engine.POST("/command/", handlerWithStatsd(statsdClient, masterCommandHandler))
	engine.POST("/register/", masterRegisterHandlerFactory(statsdAddr, targetUrl))
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
		log.Error().Err(err).Msg("Error generating request")
		return
	}

	client := getDefaultHttpClient()

	for _, workerUrl := range workerUrls {
		go func(workerUrl string) {
			body := bytes.NewReader(newParams)
			var commandUrl = fmt.Sprintf("%s/command/", workerUrl)
			req, err := http.NewRequest("POST", commandUrl, body)
			if err != nil {
				log.Error().Err(err).Msgf("could not create request for url: `%s`", workerUrl)
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				log.Error().Err(err).Msgf(" error sending command to client '%s'", workerUrl)
			}
			if resp != nil {
				err = resp.Body.Close()
				if err != nil {
					log.Error().Err(err).Msg("error closing the body of the attack")
				}
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
					err = resp.Body.Close()
					if err != nil {
						log.Error().Err(err).Msg("could not close response body")
					}
				}
			}()
			if err != nil || (resp != nil && resp.StatusCode > 300) {
				log.Error().Err(err).Msgf("Worker %s did not respond to ping", workerUrl)
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
	globalMasterMetrics.desiredRate = 0
	for _, worker := range workerUrls {
		go func(workerUrl string) {
			var stopUrl = fmt.Sprintf("%s/stop/", workerUrl)
			var resp, err = client.Get(stopUrl)
			if err != nil {
				log.Error().Err(err).Msgf("Could not send request to client %s", workerUrl)
				return
			}
			defer func() {
				err = resp.Body.Close()
				if err != nil {
					log.Error().Err(err).Msg("Failed to close body form close response")
				}
			}()

		}(worker)
	}
	ctx.JSON(http.StatusOK, okJsonResponse())
}

func masterCommandHandler(statsdClient *statsd.Client, ctx *gin.Context) {
	var params tests.TestParams
	log.Info().Msg("command handler called")
	if err := ctx.ShouldBindJSON(&params); err != nil {
		log.Error().Err(err).Msgf("Could not parse command:\n%v", params)
		ctx.JSON(http.StatusBadRequest, "Could not parse command")
		return
	}
	freq, err := utils.PerSecond(int64(params.NumMessages), params.Per)
	if err != nil {
		log.Error().Msgf("Failed to calculate request frequency for %d per %v", params.NumMessages, params.Per)
		return
	}
	globalMasterMetrics.desiredRate = freq
	go ForwardAttack(params) // no need to wait for sending it to clients
	ctx.JSON(http.StatusOK, "Attack forwarded to workers")
}

func masterRegisterHandlerFactory(statsdClient string, targetUrl string) func(*gin.Context) {
	return func(ctx *gin.Context) {
		var workerReq registerWorkerRequest
		if err := ctx.ShouldBindJSON(&workerReq); err == nil {
			addWorker(workerReq.WorkerUrl)
			ctx.JSON(http.StatusOK, sendServerConfig(targetUrl, statsdClient))
		} else {
			log.Error().Err(err).Msg("Error while trying to register worker")
			ctx.JSON(http.StatusBadRequest, errorJsonResponse("Could not parse registration request"))
		}
	}
}

func masterUnregisterHandler(ctx *gin.Context) {
	var workerReq registerWorkerRequest
	if err := ctx.ShouldBindJSON(&workerReq); err == nil {
		removeWorker(workerReq.WorkerUrl)
		ctx.JSON(http.StatusOK, okJsonResponse())
	} else {
		log.Error().Err(err).Msg("Error while trying to unregister worker")
		ctx.JSON(http.StatusBadRequest, errorJsonResponse("Could not register request"))
	}
}
func mainDocsHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "docs.html", gin.H{"content": getDocContent()})
}

func getDocContent() string {
	fileName := "_Documents/TestFormat.md"
	content, err := os.ReadFile(fileName)
	if err != nil {
		log.Error().Err(err).Msgf("could not read doc file:%s ", fileName)
		return "<h1>Internal Error</h1>"
	}
	return string(content)
}
