// Package server contains all server logic
package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"

	"github.com/stsg/gophkeeper/pkg/config"
	"github.com/stsg/gophkeeper/pkg/status"
	postgres "github.com/stsg/gophkeeper/pkg/store"
)

type Rest struct {
	Listen   string
	Version  string
	Status   Status
	Config   *config.Parameters
	Timeout  time.Duration
	Store    *postgres.Storage
	Secret   []byte
	LifeSpan time.Duration
}

type Status interface {
	Get() (*status.Info, error)
}

// Run starts the HTTP server and listens for incoming requests.
//
// It takes a context.Context as a parameter.
// Returns an error.
func (s *Rest) Run(ctx context.Context) error {
	log.Printf("[INFO] start http server on %s", s.Listen)

	httpServer := &http.Server{
		Addr:              s.Listen,
		Handler:           s.router(),
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       time.Second,
		ErrorLog:          log.ToStdLogger(log.Default(), "WARN"),
	}

	go func() {
		<-ctx.Done()
		if httpServer != nil {
			if err := httpServer.Close(); err != nil {
				log.Printf("[ERROR] failed to close http server: %v", err)
			}
		}
	}()

	return httpServer.ListenAndServe()
}

func (s *Rest) router() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RealIP, rest.Recoverer(log.Default()))
	router.Use(rest.Throttle(100), middleware.Timeout(60*time.Second))
	router.Use(rest.AppInfo("gophkeeper", "sartorus", s.Version))
	router.Use(rest.Ping)
	router.Use(logger.New(logger.Log(log.Default()), logger.WithBody, logger.Prefix("[DEBUG]")).Handler)
	router.Use(rest.Gzip("application/json", "text/html"))
	router.Use(middleware.Compress(5, "application/json", "text/html"))
	router.Use(rest.BasicAuth(s.Auth))

	router.Route("/", func(r chi.Router) {
		r.Get("/echo", s.echo)
		r.Get("/status", s.status)
		r.Post("/register", s.Register)
		r.Post("/login", s.Login)
		r.Mount("/vault", s.VaultRoute())
	})

	return router
}

func (s *Rest) echo(w http.ResponseWriter, r *http.Request) {
	echo := struct {
		Message    string            `json:"message"`
		Request    string            `json:"request"`
		Host       string            `json:"host"`
		Headers    map[string]string `json:"headers"`
		RemoteAddr string            `json:"remote_addr"`
	}{
		Message:    "eChO <---> EcHo",
		Request:    r.Method + " " + r.RequestURI,
		Host:       r.Host,
		Headers:    make(map[string]string),
		RemoteAddr: r.RemoteAddr,
	}

	for k, vv := range r.Header {
		echo.Headers[k] = strings.Join(vv, "; ")
	}

	rest.RenderJSON(w, &echo)
}

func (s *Rest) status(w http.ResponseWriter, r *http.Request) {
	info, err := s.Status.Get()
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to get status")
		return
	}
	rest.RenderJSON(w, info)
}

func (s *Rest) Auth(login string, password string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	user, error := s.Store.GetIdentity(ctx, login)
	if error != nil {
		log.Printf("[ERROR] failed to get user: %v", error)
		return false
	}

	if user.Passw != password {
		log.Printf("[ERROR] wrong password: %v", error)
		return false
	}

	return true
}
