package httpapi

import (
	"net/http"
	"time"

	"email-subscription-service/internal/config"
	"email-subscription-service/internal/httpapi/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(cfg config.Config, subSvc handlers.SubscriptionService) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/health", handlers.Health)

	h := handlers.SubscriptionHandlers{Svc: subSvc}

	// Swagger basePath: /api
	r.Route("/api", func(r chi.Router) {
		r.Post("/subscribe", h.Subscribe)                 // POST /api/subscribe
		r.Get("/confirm/{token}", h.ConfirmSubscription)  // GET /api/confirm/{token}
		r.Get("/unsubscribe/{token}", h.Unsubscribe)      // GET /api/unsubscribe/{token}
		r.Get("/subscriptions", h.GetSubscriptions)       // GET /api/subscriptions?email=
	})

	return r
}