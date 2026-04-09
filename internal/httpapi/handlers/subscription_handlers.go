package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"email-subscription-service/internal/domain"

	"github.com/go-chi/chi/v5"
)

type SubscriptionHandlers struct {
	Svc SubscriptionService
}

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

	err = h.Svc.Subscribe(r.Context(), email, repo)
	if err != nil {
		writeSwaggerError(w, err)
		return
	}
	writeJSONOK(w)
}

func (h SubscriptionHandlers) ConfirmSubscription(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	err := h.Svc.Confirm(r.Context(), token)
	if err != nil {
		writeSwaggerError(w, err)
		return
	}
	writeJSONOK(w)
}

func (h SubscriptionHandlers) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	err := h.Svc.Unsubscribe(r.Context(), token)
	if err != nil {
		writeSwaggerError(w, err)
		return
	}
	writeJSONOK(w)
}

func (h SubscriptionHandlers) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	subs, err := h.Svc.ListByEmail(r.Context(), email)
	if err != nil {
		writeSwaggerError(w, err)
		return
	}

	if subs == nil {
		subs = []domain.Subscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(subs)
}

func writeJSONOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func writeSwaggerError(w http.ResponseWriter, err error) {
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func parseSubscribeInput(r *http.Request) (email string, repo string, err error) {
	ct := r.Header.Get("Content-Type")
	// JSON support: Content-Type may include charset.
	if strings.HasPrefix(ct, "application/json") {
		var req subscribeRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			return "", "", err
		}
		// Optional: ensure there isn't trailing garbage
		if dec.More() {
			return "", "", errors.New("invalid json body")
		}
		return req.Email, req.Repo, nil
	}
	// Default: keep swagger formData support
	if err := r.ParseForm(); err != nil {
		return "", "", err
	}
	return r.FormValue("email"), r.FormValue("repo"), nil
}