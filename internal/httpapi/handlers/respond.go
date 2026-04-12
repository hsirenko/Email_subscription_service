package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"email-subscription-service/internal/domain"

	"github.com/go-chi/chi/v5/middleware"
)

func writeJSONOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// writeSwaggerError maps domain errors to HTTP statuses from swagger.yaml;
// any other error becomes 500 and is logged for operators.
func writeSwaggerError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidRepoFormat),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidToken):
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	case errors.Is(err, domain.ErrRepoNotFound),
		errors.Is(err, domain.ErrTokenNotFound):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	case errors.Is(err, domain.ErrAlreadySubscribed):
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
	default:
		reqID := ""
		if r != nil {
			reqID = middleware.GetReqID(r.Context())
		}
		if reqID != "" {
			log.Printf("api error -> 500: request_id=%s err=%v", reqID, err)
		} else {
			log.Printf("api error -> 500: %v", err)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
