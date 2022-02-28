package web_server

import (
	"fmt"
	vegeta "github.com/tsenart/vegeta/lib"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/getsentry/go-load-tester/tests"
)

func RunWebServer(port string) {

	paramChannel := make(chan tests.TestParams)
	defer close(paramChannel)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/stop/", withParamChannel(paramChannel, stopHandler))
	engine.POST("/stop/", withParamChannel(paramChannel, stopHandler))
	engine.POST("/command/", withParamChannel(paramChannel, commandHandler))
	go worker(paramChannel)
	if len(port) > 0 {
		port = fmt.Sprintf(":%s", port)
	}
	//log.Fatal(http.ListenAndServe(port, nil))
	_ = engine.SetTrustedProxies([]string{})
	_ = engine.Run(port)
}

type handlerWithCommand func(chan<- tests.TestParams, *gin.Context)

func withParamChannel(paramsChannel chan<- tests.TestParams, handler handlerWithCommand) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(paramsChannel, ctx)
	}
}

func stopHandler(params chan<- tests.TestParams, ctx *gin.Context) {
	params <- tests.TestParams{} // send a "0" params will be interpreted as a "Stop request"
	ctx.String(200, "Stopping requested")
}

func commandHandler(cmd chan<- tests.TestParams, ctx *gin.Context) {
	var params tests.TestParams
	err := ctx.BindJSON(&params)
	if err != nil {
		ctx.String(400, "Could not parse body")
	}
	cmd <- params
	ctx.String(200, "Command Accepted")

}

func createTargeter(params tests.TestParams) vegeta.Targeter {
	if params.AttackDuration == 0 {
		log.Printf("Zero attack duration, stopping")
		return nil
	}
	targeterBuilder := tests.GetTargeter(params.Name)
	if targeterBuilder == nil {
		log.Printf("Invalid attack type %s", params.Name)
		return nil
	}
	return targeterBuilder("TODO-the_url", params.Params)

}

func worker(paramsChan <-chan tests.TestParams) {
	var targeter vegeta.Targeter
	var params tests.TestParams

	for {
	attack:
		select {
		case params = <-paramsChan:
			targeter = createTargeter(params)
		default:
			if targeter != nil {
				var metrics vegeta.Metrics
				rate := vegeta.Rate{Freq: params.NumMessages, Per: params.Per}
				attacker := vegeta.NewAttacker(vegeta.Timeout(time.Millisecond*500), vegeta.Redirects(0))
				for res := range attacker.Attack(targeter, rate, params.AttackDuration, "TODO some name for the attack") {
					metrics.Add(res)
					select {
					case params = <-paramsChan:
						targeter = createTargeter(params)
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
