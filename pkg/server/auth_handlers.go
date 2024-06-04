package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	postgres "github.com/stsg/gophkeeper/pkg/store"
)

// Register handles the registration of a new user.
//
// It expects a POST request with a JSON payload containing the user's credentials.
// The payload should have the following structure:
//
//	{
//	  "username": "string",
//	  "password": "string"
//	}
//
// If the request payload is invalid or missing required fields, it returns a 400 Bad Request response.
// If the user already exists, it returns a 409 Conflict response.
// If there is an error during the registration process, it returns a 500 Internal Server Error response.
// If the registration is successful, it returns a 200 OK response.
//
// Parameters:
// - w: http.ResponseWriter - the response writer used to send the response
// - r: *http.Request - the incoming request
//
// Returns: None
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

// Login handles the login functionality for the REST API.
//
// It expects a POST request with a JSON payload containing the user's credentials.
// The payload should have the following structure:
//
//	{
//	  "username": "string",
//	  "password": "string"
//	}
//
// If the request payload is invalid or missing required fields, it returns a 400 Bad Request response.
// If the user does not exist or the password is incorrect, it returns a 401 Unauthorized response.
// If there is an error during the authentication process, it returns a 500 Internal Server Error response.
// If the authentication is successful, it returns a 200 OK response with the authentication token in the Authorization header.
//
// Parameters:
// - w: http.ResponseWriter - the response writer used to send the response
// - r: *http.Request - the incoming request
//
// Returns: None
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
