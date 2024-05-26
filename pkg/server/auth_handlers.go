package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	postgres "github.com/stsg/gophkeeper/pkg/store"
)

type Creds struct {
	Login string `json:"username"`
	Passw string `json:"password"`
}

func (s *Rest) Register(w http.ResponseWriter, r *http.Request) {
	var rBody map[string]any
	if err := json.NewDecoder(r.Body).Decode(&rBody); err != nil {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s RegisterHook", reqID)

	var c postgres.Creds

	if val, ok := rBody["username"].(string); ok {
		c.Login = val
	} else {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	if val, ok := rBody["password"].(string); ok {
		c.Passw = val
	} else {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	err := s.Store.Register(ctx, c)

	if err != nil {
		if errors.Is(err, postgres.ErrUniqueViolation) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		if errors.Is(err, postgres.ErrNoExists) {
			w.WriteHeader(http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] login %s registered RegisterHook", c.Login)
	w.WriteHeader(http.StatusOK)
}

func (s *Rest) Login(w http.ResponseWriter, r *http.Request) {
	var rBody map[string]any
	if err := json.NewDecoder(r.Body).Decode(&rBody); err != nil {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s LoginHook", reqID)

	var c postgres.Creds

	if val, ok := rBody["username"].(string); ok {
		c.Login = val
	} else {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	if val, ok := rBody["password"].(string); ok {
		c.Passw = val
	} else {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	token, err := s.Store.Authenticate(ctx, c)
	if err != nil {
		if errors.Is(err, postgres.ErrUniqueViolation) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] login %s logged LoginHook", c.Login)
	w.Header().Set("Authorization", token)
	w.WriteHeader(http.StatusOK)
}
