package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type subscribeRequest struct {
	Email string `json:"email"`
	Repo  string `json:"repo"`
}

func (h SubscriptionHandlers) Subscribe(w http.ResponseWriter, r *http.Request) {
	email, repo, err := parseSubscribeInput(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := h.Svc.Subscribe(r.Context(), email, repo); err != nil {
		writeSwaggerError(w, r, err)
		return
	}
	writeJSONOK(w)
}

func parseSubscribeInput(r *http.Request) (email string, repo string, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var req subscribeRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			return "", "", err
		}
		if dec.More() {
			return "", "", errors.New("invalid json body")
		}
		return req.Email, req.Repo, nil
	}
	if err := r.ParseForm(); err != nil {
		return "", "", err
	}
	return r.FormValue("email"), r.FormValue("repo"), nil
}
