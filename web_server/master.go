package web_server

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
)

/*
Contains code for the Master web server
*/

var masterState struct {
	lock    sync.Mutex
	workers []string // urls of registered workers
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
}

func removeWorker(url string) {
	masterState.lock.Lock()
	defer masterState.lock.Unlock()
	l := len(masterState.workers)

	var x int
	var a = make([]int, 7)
	x, a = a[len(a)-1], a[:len(a)-1]
	fmt.Printf("%d", x)

	for idx, workerUrl := range masterState.workers {
		if workerUrl == url {
			masterState.workers[idx] = masterState.workers[l-1]
			masterState.workers = masterState.workers[:l-1]
			return
		}
	}
}

func RunMasterWebServer(port string) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()

	engine.GET("/stop/", masterStopHandler)
	engine.POST("/stop/", masterStopHandler)
	engine.POST("/command/", masterCommandHandler)
	engine.POST("/register/", masterRegisterHandler)
	engine.POST("/unregister/", masterUnregisterHandler)
	if len(port) > 0 {
		port = fmt.Sprintf(":%s", port)
	}
	_ = engine.SetTrustedProxies([]string{})
	_ = engine.Run(port)
}

func masterStopHandler(ctx *gin.Context) {

}

func masterCommandHandler(ctx *gin.Context) {

}

func masterRegisterHandler(ctx *gin.Context) {

}

func masterUnregisterHandler(ctx *gin.Context) {

}
