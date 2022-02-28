package web_server

import (
	"fmt"
	vegeta "github.com/tsenart/vegeta/lib"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/getsentry/go-load-tester/tests"
)

func RunWebServer(port string, targetUrl string) {

	paramChannel := make(chan tests.TestParams)
	defer close(paramChannel)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/stop/", withParamChannel(paramChannel, stopHandler))
	engine.POST("/stop/", withParamChannel(paramChannel, stopHandler))
	engine.POST("/command/", withParamChannel(paramChannel, commandHandler))
	go worker(targetUrl, paramChannel)
	if len(port) > 0 {
		port = fmt.Sprintf(":%s", port)
	}
	_ = engine.SetTrustedProxies([]string{})
	_ = engine.Run(port)
}

type handlerWithCommand func(chan<- tests.TestParams, *gin.Context)

// withParamChannel constructs a Gin handler from a handler that also accepts a command channel
func withParamChannel(paramsChannel chan<- tests.TestParams, handler handlerWithCommand) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(paramsChannel, ctx)
	}
}

// stopHandler handle stop requests
func stopHandler(params chan<- tests.TestParams, ctx *gin.Context) {
	params <- tests.TestParams{} // send a "0" params will be interpreted as a "Stop request"
	ctx.String(200, "Stopping requested")
}

// commandHandler handle command requests
func commandHandler(cmd chan<- tests.TestParams, ctx *gin.Context) {
	var params tests.TestParams
	err := ctx.BindJSON(&params)
	if err != nil {
		ctx.String(400, "Could not parse body")
	}
	cmd <- params
	ctx.String(200, "Command Accepted")

}

// createTargeter creates a targeter for the passed test parameters
func createTargeter(targetUrl string, params tests.TestParams) vegeta.Targeter {
	if params.AttackDuration == 0 {
		log.Printf("Zero attack duration, stopping")
		return nil
	}
	targeterBuilder := tests.GetTargeter(params.Name)
	if targeterBuilder == nil {
		log.Printf("Invalid attack type %s", params.Name)
		return nil
	}
	return targeterBuilder(targetUrl, params.Params)

}

// worker the worker that handles Vegeta attacks
//
// The worker uses a command channel to accept new commands
// Once a command is received the current attack (if there is a current attack)
// is stopped and a new attack started
func worker(targetUrl string, paramsChan <-chan tests.TestParams) {
	var targeter vegeta.Targeter
	var params tests.TestParams

	for {
	attack:
		select {
		case params = <-paramsChan:
			targeter = createTargeter(targetUrl, params)
		default:
			if targeter != nil {
				var metrics vegeta.Metrics
				rate := vegeta.Rate{Freq: params.NumMessages, Per: params.Per}
				attacker := vegeta.NewAttacker(vegeta.Timeout(time.Millisecond*500), vegeta.Redirects(0))
				for res := range attacker.Attack(targeter, rate, params.AttackDuration, params.Description) {
					metrics.Add(res)
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
				metrics.Close()             //TODO do something with the metrics or don't collect them
				time.Sleep(1 * time.Second) // sleep a bit, so we don't busy spin when there is no attack
			}
		}
	}
}
