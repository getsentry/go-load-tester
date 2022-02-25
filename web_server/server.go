/*
Copyright © 2021 Sentry

*/
package web_server

import (
	"fmt"
	vegeta "github.com/tsenart/vegeta/lib"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/getsentry/go-load-tester/tests"
)

type Command int32

const (
	Stop    Command = 0
	Exit    Command = 1
	DoStuff Command = 2
)

func RunWebServer(port string) {
	cmdChannel := make(chan Command)
	defer close(cmdChannel)
	paramChannel := make(chan tests.TestParams)
	defer close(paramChannel)
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/command/:commandId/*rest", withCommandChannel(cmdChannel, commandHandler))
	go worker(cmdChannel, paramChannel)
	if len(port) > 0 {
		port = fmt.Sprintf(":%s", port)
	}
	//log.Fatal(http.ListenAndServe(port, nil))
	engine.SetTrustedProxies([]string{})
	engine.Run(port)
}

type handlerWithCommand func(chan<- Command, *gin.Context)

func withCommandChannel(command chan<- Command, handler handlerWithCommand) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		handler(command, ctx)
	}
}

//func commandHandler(w http.ResponseWriter, req *http.Request) {
func commandHandler(cmd chan<- Command, ctx *gin.Context) {
	command := ctx.Param("commandId")
	rest := ctx.Param("rest")
	switch command {
	case "stop":
		log.Println("Sending stop request")
		cmd <- Stop
	case "do":
		log.Println("Start doing stuff")
		cmd <- DoStuff
	default:
		log.Printf("Received command %s, ignoring it for now", command)
	}
	ctx.String(200, "Hello, from command: %s, %s", command, rest)

}

func worker(cmd <-chan Command, parameters <-chan tests.TestParams) {
	for {
		select {
		case command := <-cmd:
			switch command {
			case Exit:
				return
			case Stop:
				log.Println("Stop sent while not doing anything")
				continue // nothing to do here we were not in a loop
			case DoStuff:
			doStuff:
				for i := 1; i < 10; i++ {
					log.Printf("Doing some stuff here loop: %d of 10\n", i)
					select {
					case <-time.After(5 * time.Second):
						continue
					case command = <-cmd:
						if command == Exit {
							return
						}
						if command == Stop {
							// stopping
							log.Println("Stop called")
							break doStuff
						}
						if command == DoStuff {
							log.Printf("Do stuff called we should re initialize")
							continue
						}
					}
				}
				log.Println("Finished doing stuff, taking a break")
			}
		}

	}
}

func createTargeter(params tests.TestParams) vegeta.Targeter {
	targeterBuilder := tests.GetTargeter(params.Name)
	if targeterBuilder == nil {
		log.Printf("Invalid attack type %s", params.Name)
		return nil
	}
	return targeterBuilder("TODO-the_url", params.Params)

}

func worker2(timingChan <-chan tests.TimingParams, paramsChan <-chan tests.TestParams) {
	var timing *tests.TimingParams
	var targeter vegeta.Targeter
	attaker := vegeta.NewAttacker(vegeta.Timeout(time.Millisecond*500), vegeta.Redirects(0))
	var params tests.TestParams

	for {
		select {
		case timing = <-timingChan:
		case params = <-paramsChan:
			targeter = createTargeter(params)
		default:
			if timing != nil && targeter != nil {
				var metrics vegeta.Metrics
				rate := vegeta.Rate{Freq: timing.NumMessages, Per: timing.Per}
			attack:
				for res := range attaker.Attack(targeter, rate, timing.AttackDuration, "TODO some name for the attack") {
					metrics.Add(res)
					select {
					case timing = <-timingChan:
						break attack // starts a new attack
					case params = <-paramsChan:
						targeter = createTargeter(params)
						break attack // starts a new attack
					default:
						continue
					}
				}
				// finish current attack reset timing
				timing = nil
				metrics.Close()             //TODO do something with the metrics or don't collect them
				time.Sleep(1 * time.Second) // sleep a bit so we don't busy spin when there is no attack
			}
		}

	}

}

/*
Worker structure (just pseudo code for now)

func worker2(paceChan <-chan Pace, paramsChan <-chan TestParams) {
	var pace *Pace
	var params *TestParams
	for {
		if pace != nil && params != nil {
		attackLoop:
			for res := attacker.Attack(pace, params) {
				// process response
				select {
				case params2 := <-paramsChan:
				//pass p2 to the new targeter
				case pace2 := <-paceChan:
					if pace2 == STOP {
						pace = nil
					} else {
						pace = pace2
					}
					break attackLoop
				default:
					continue // just continue the attack
				}
			}
		}
	}

}
*/
