package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"

	postgres "github.com/stsg/gophkeeper/pkg/store"
)

func (s *Rest) VaultRoute() http.Handler {
	router := chi.NewRouter()
	router.Mount("/piece", s.VaultPieceRoute())
	router.Mount("/blob", s.VaultBlobRoute())
	router.Get("/", s.VaultList)
	router.Delete("/{rid}", s.VaultDelete)
	return router
}

func (s *Rest) VaultList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s VaultListHook", reqID)

	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(ctx, token)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			status = http.StatusUnauthorized
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	resources, err := s.Store.List(ctx, creds)
	if err != nil {
		var status = http.StatusInternalServerError
		http.Error(w, http.StatusText(status), status)
		return
	}

	var response = make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		response = append(
			response,
			map[string]any{
				"rid":  (int64)(resource.ID),
				"meta": resource.Meta,
				"type": (int)(resource.Type),
			},
		)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		log.Printf("[ERROR] failed to write response: %s\n", err.Error())
	}
}

func (s *Rest) VaultDelete(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s VaultDeleteHook", reqID)

	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(ctx, token)
	if err != nil {
		var status = http.StatusInternalServerError
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			status = http.StatusUnauthorized
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	var rid, ridError = strconv.Atoi(chi.URLParam(r, "rid"))
	if ridError != nil {
		status := http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	if err := s.Store.Delete(ctx, postgres.ResourceID(rid), creds); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, postgres.ErrResourceNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Rest) VaultPieceRoute() http.Handler {
	router := chi.NewRouter()
	router.Put("/", s.VaultPieceEncrypt)
	router.Get("/{rid}", s.VaultPieceDecrypt)
	return router
}

func (s *Rest) VaultPieceEncrypt(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s VaultDeleteHook", reqID)

	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(ctx, token)
	if err != nil {
		var status = http.StatusInternalServerError
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			status = http.StatusUnauthorized
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	var request struct {
		Meta    string `json:"meta"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}
	var content = make([]byte, len(request.Content))
	if _, err := base64.RawStdEncoding.Decode(content, ([]byte)(request.Content)); err != nil {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	var piece = postgres.Piece{
		Meta:    request.Meta,
		Content: content,
	}
	var password = r.Header.Get("X-Password")
	if password == "" {
		var status = http.StatusUnauthorized
		http.Error(w, http.StatusText(status), status)
		return
	}
	rid, err := s.Store.StorePiece(ctx, piece, creds)
	if err != nil {
		var status = http.StatusInternalServerError
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			status = http.StatusUnauthorized
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.WriteHeader(http.StatusCreated)
	var response struct {
		RID int64 `json:"rid"`
	}
	response.RID = (int64)(rid)
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		log.Printf("Failed to write response: %s", err.Error())
	}
}

func (s *Rest) VaultPieceDecrypt(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s VaultPieceDecryptHook", reqID)

	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(ctx, token)
	if err != nil {
		var status = http.StatusInternalServerError
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			status = http.StatusUnauthorized
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		var status = http.StatusUnauthorized
		http.Error(w, http.StatusText(status), status)
		return
	}

	rid, err := strconv.Atoi(chi.URLParam(r, "rid"))
	if err != nil {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	piece, err := s.Store.RestorePiece(ctx, (postgres.ResourceID)(rid), creds)
	if err != nil {
		var status = http.StatusInternalServerError
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			status = http.StatusUnauthorized
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	var response struct {
		Meta    string `json:"meta"`
		Content string `json:"content"`
	}
	response.Meta = piece.Meta
	response.Content = base64.RawStdEncoding.EncodeToString(
		bytes.ReplaceAll(
			piece.Content,
			[]byte{'\x00'},
			[]byte{},
		),
	)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to write response: %s", err.Error())
	}
}

func (s *Rest) VaultBlobRoute() http.Handler {
	// TODO: implement me
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusNotImplemented, fmt.Errorf("not yet implemented"), "not yet implemented")
	})
}
