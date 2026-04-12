// Command api runs the HTTP server, Postgres migrations, GitHub client, mailer,
// and the background release scanner in one process.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"email-subscription-service/internal/config"
	"email-subscription-service/internal/httpapi"
	"email-subscription-service/internal/integrations/email"
	emailLog "email-subscription-service/internal/integrations/email/log"
	emailSMTP "email-subscription-service/internal/integrations/email/smtp"
	"email-subscription-service/internal/integrations/github"
	"email-subscription-service/internal/jobs"
	"email-subscription-service/internal/service"
	"email-subscription-service/internal/store/postgres"

	mgpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// "api error -> 500: ...". Send default logs to stdout so they appear with access logs.
	log.SetOutput(os.Stdout)

	cfg := config.FromEnv()

	gh := github.NewHTTPClient(github.HTTPClientConfig{
		Token: cfg.GitHubToken,
	})
	var mailer email.Sender
	switch cfg.EmailDriver {
	case "smtp":
		port, err := strconv.Atoi(cfg.SMTPPort)
		if err != nil {
			log.Fatalf("invalid SMTP_PORT %q: %v", cfg.SMTPPort, err)
		}
		mailer = emailSMTP.New(emailSMTP.Config{
			Host:     cfg.SMTPHost,
			Port:     port,
			Username: cfg.SMTPUser,
			Password: cfg.SMTPPass,
			From:     cfg.SMTPFrom,
		})
	case "log", "":
		mailer = emailLog.Sender{}
	default:
		log.Fatalf("unknown EMAIL_DRIVER %q (expected log or smtp)", cfg.EmailDriver)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	driver, err := mgpostgres.WithInstance(db, &mgpostgres.Config{})
	if err != nil {
		log.Fatalf("migrate driver: %v", err)
	}
	if err := postgres.RunMigrations(driver); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	st := &postgres.Store{DB: db}

	subSvc := service.SubscriptionService{
		Store:     st,
		GitHub:    gh,
		Email:     mailer,
		PublicURL: cfg.PublicURL,
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           httpapi.NewRouter(cfg, subSvc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	scanner := jobs.Scanner{
		Store:   st, // <- Postgres store implements ReleaseJobStore
		GitHub:  gh,
		Email:   mailer,
		Every:   cfg.ScanInterval,
		BaseURL: cfg.PublicURL,
	}
	go scanner.Run(ctx)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}

	log.Printf("shutdown complete")
}
