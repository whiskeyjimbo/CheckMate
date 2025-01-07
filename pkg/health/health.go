package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	StatusUp       = "UP"
	StatusReady    = "READY"
	StatusNotReady = "NOT_READY"
)

type Response struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

var isReady atomic.Bool

func SetReady(ready bool) {
	isReady.Store(ready)
}

func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Status:    StatusUp,
		Timestamp: time.Now(),
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Timestamp: time.Now(),
	}

	if isReady.Load() {
		response.Status = StatusReady
		writeJSONResponse(w, http.StatusOK, response)
		return
	}

	response.Status = StatusNotReady
	writeJSONResponse(w, http.StatusServiceUnavailable, response)
}

func writeJSONResponse(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
