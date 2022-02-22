/*
Copyright © 2021 Sentry

*/
package web_server

import (
	"github.com/gin-gonic/gin"
	"log"
	"time"
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
	engine := gin.Default()
	engine.GET("/command/:commandId/*rest", withCommandChannel(cmdChannel, commandHandler))
	go worker(cmdChannel)
	engine.Run()
	//http.HandleFunc("/command/", commandHandler)
	//if len(port) > 0 {
	//	port = fmt.Sprintf(":%s", port)
	//}
	//log.Fatal(http.ListenAndServe(port, nil))
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

func worker(cmd <-chan Command) {
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
