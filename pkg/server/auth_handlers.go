package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	postgres "github.com/stsg/gophkeeper/pkg/store"
)

func (s *Rest) Register(w http.ResponseWriter, r *http.Request) {
	var cr postgres.Creds
	if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s RegisterHook", reqID)

	if cr.Login == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if cr.Passw == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err := s.Store.Register(r.Context(), cr)

	if err != nil {
		if errors.Is(err, postgres.ErrUniqueViolation) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		if errors.Is(err, postgres.ErrNoExists) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] login %s registered RegisterHook", cr.Login)
	w.WriteHeader(http.StatusOK)
}

func (s *Rest) Login(w http.ResponseWriter, r *http.Request) {
	var cr postgres.Creds
	if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s LoginHook", reqID)

	if cr.Login == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if cr.Passw == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	token, err := s.Store.Authenticate(r.Context(), cr)
	if err != nil {
		if errors.Is(err, postgres.ErrUniqueViolation) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] login %s logged LoginHook", cr.Login)
	w.Header().Set("Authorization", token)
	w.WriteHeader(http.StatusOK)
}
