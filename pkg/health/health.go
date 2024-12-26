package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Response struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

var isReady atomic.Bool

func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Status:    "UP",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := Response{
		Timestamp: time.Now(),
	}

	if isReady.Load() {
		response.Status = "READY"
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Status = "NOT_READY"
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(response)
}

func SetReady(ready bool) {
	isReady.Store(ready)
}
