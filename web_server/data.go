package web_server

import "github.com/gin-gonic/gin"

// registerWorkerRequest is the body of the register http request sent by a worker
// to register to a master.
type registerWorkerRequest struct {
	WorkerUrl string `json:"workerUrl"`
}

type configParams struct {
	TargetUrl       string `json:"targetUrl,omitempty"`
	StatsdServerUrl string `json:"statsdServerUrl,omitempty"`
}
type registerWorkerResponse struct {
	Error  string       `json:"error,omitempty"`
	Status string       `json:"status,omitempty"`
	Params configParams `json:"params,omitempty"`
}

func sendServerConfig(targetUrl string, statsdServerUrl string) interface{} {
	return registerWorkerResponse{
		Status: "ok",
		Params: configParams{
			TargetUrl:       targetUrl,
			StatsdServerUrl: statsdServerUrl,
		},
	}
}

func okJsonResponse() interface{} {
	return gin.H{"status": "ok"}
}

func errorJsonResponse(errorMessage string) registerWorkerResponse {
	return registerWorkerResponse{Error: errorMessage}
}
