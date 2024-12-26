package health

import (
	"encoding/json"
	"net/http"
	"time"
)

type Response struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Status:    "UP",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Status:    "READY",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
