package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"email-subscription-service/internal/config"
	"email-subscription-service/internal/httpapi/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(cfg config.Config, subSvc handlers.SubscriptionService) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	if co := buildCORS(cfg); co != nil {
		r.Use(cors.Handler(*co))
	}
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

func buildCORS(cfg config.Config) *cors.Options {
	if len(cfg.CORSAllowedOrigins) == 0 && !cfg.CORSAllowVercelSubdomains {
		return nil
	}
	base := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           600,
	}
	if cfg.CORSAllowVercelSubdomains {
		extra := cfg.CORSAllowedOrigins
		base.AllowOriginFunc = func(_ *http.Request, origin string) bool {
			return corsOriginAllowed(origin, extra)
		}
		return &base
	}
	base.AllowedOrigins = cfg.CORSAllowedOrigins
	return &base
}

func corsOriginAllowed(origin string, extra []string) bool {
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, ".vercel.app") {
		return true
	}
	for _, a := range extra {
		if a == origin {
			return true
		}
	}
	return false
}