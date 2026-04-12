package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"email-subscription-service/internal/domain"

	"github.com/go-chi/chi/v5"
)

// SubscriptionServicer is the application boundary used by HTTP handlers.
// Implemented by internal/service.SubscriptionService (naming differs to avoid clashing with that struct type in tooling).
type SubscriptionServicer interface {
	Subscribe(ctx context.Context, email string, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	ListByEmail(ctx context.Context, email string) ([]domain.Subscription, error)
}

// SubscriptionHandlers wires HTTP routes to a SubscriptionServicer implementation.
type SubscriptionHandlers struct {
	Svc SubscriptionServicer
	// WebUIURL is the static subscribe UI (e.g. Vercel). If empty, a documented default is used for links.
	WebUIURL string
	// APIPublicURL is the public API base (PUBLIC_URL) for links to this service, e.g. /api/confirm/thanks.
	APIPublicURL string
}

func (h SubscriptionHandlers) ConfirmSubscription(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if err := h.Svc.Confirm(r.Context(), token); err != nil {
		if wantsJSONResponse(r) {
			writeSwaggerError(w, r, err)
			return
		}
		writeConfirmErrorHTMLFromErr(w, h, err)
		return
	}
	if wantsJSONResponse(r) {
		writeJSONOK(w)
		return
	}
	h.writeConfirmSuccessHTML(w)
}

func (h SubscriptionHandlers) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if err := h.Svc.Unsubscribe(r.Context(), token); err != nil {
		if wantsJSONResponse(r) {
			writeSwaggerError(w, r, err)
			return
		}
		writeUnsubscribeErrorHTMLFromErr(w, h, err)
		return
	}
	if wantsJSONResponse(r) {
		writeJSONOK(w)
		return
	}
	h.writeUnsubscribeSuccessHTML(w)
}

func (h SubscriptionHandlers) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	subs, err := h.Svc.ListByEmail(r.Context(), email)
	if err != nil {
		writeSwaggerError(w, r, err)
		return
	}

	if subs == nil {
		subs = []domain.Subscription{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(subs)
}
