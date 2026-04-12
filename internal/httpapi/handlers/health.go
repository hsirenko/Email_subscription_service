package handlers

import (
	"net/http"
)

// Health responds with 200 and plain text "ok" for load balancers and probes.
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
