package server

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"

	postgres "github.com/stsg/gophkeeper/pkg/store"
)

// VaultRoute returns an http.Handler that handles the routing for the vault API.
//
// It mounts the "/piece" and "/blob" routes to their respective handlers,
// and defines GET and DELETE routes for "/" and "/{rid}" respectively.
// The handlers for these routes are defined in the VaultPieceRoute, VaultBlobRoute,
// VaultList, and VaultDelete methods of the Rest struct.
//
// Returns:
// - http.Handler: The router that handles the vault API routing.
func (s *Rest) VaultRoute() http.Handler {
	router := chi.NewRouter()
	router.Get("/", s.VaultList)
	router.Delete("/{rid}", s.VaultDelete)
	router.Mount("/piece", s.VaultPieceRoute())
	router.Mount("/blob", s.VaultBlobRoute())
	return router
}

// VaultList handles the HTTP GET request to list the resources in the vault.
//
// It expects the request to have the "Authorization" header containing a valid token.
// The function retrieves the credentials from the store using the token.
// If the credentials are not found or there is an error, it returns an appropriate HTTP error response.
//
// The function then retrieves the list of resources from the store using the credentials.
// If there is an error, it returns an HTTP internal server error response.
//
// The function constructs the response by creating a slice of postgres.Resource structs,
// with each struct containing the ID, meta, and type of a resource from the list.
//
// Finally, the function writes the response as JSON to the HTTP response writer with a status code of 200.
// If there is an error encoding the response, it logs an error message.
func (s *Rest) VaultList(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s VaultListHook", reqID)

	// TODO: add auth as middleware
	// https://github.com/stsg/gophkeeper/pull/1#discussion_r1618437264
	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(r.Context(), token)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resources, err := s.Store.List(r.Context(), creds)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var response []postgres.Resource

	for _, resource := range resources {
		response = append(
			response,
			postgres.Resource{
				ID:   resource.ID,
				Meta: resource.Meta,
				Type: resource.Type,
			},
		)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		log.Printf("[ERROR] failed to write response: %s\n", err.Error())
	}
}

// VaultDelete handles the HTTP DELETE request to delete a resource from the vault.
//
// It expects the request to have the "Authorization" header containing a valid token.
// The function retrieves the credentials from the store using the token.
// If the credentials are not found or there is an error, it returns an appropriate HTTP error response.
//
// The function then parses the "rid" parameter from the request URL and attempts to delete the resource with the corresponding ID from the store using the credentials.
// If the resource is not found or there is an error, it returns an HTTP error response.
//
// If the deletion is successful, the function writes an HTTP status code of 200 to the response.
//
// Parameters:
// - w: http.ResponseWriter - the HTTP response writer.
// - r: *http.Request - the HTTP request.
//
// Return type: None.
func (s *Rest) VaultDelete(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s VaultDeleteHook", reqID)

	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(r.Context(), token)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var rid, ridError = strconv.Atoi(chi.URLParam(r, "rid"))
	if ridError != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := s.Store.Delete(r.Context(), postgres.ResourceID(rid), creds); err != nil {
		if errors.Is(err, postgres.ErrResourceNotFound) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// VaultPieceRoute returns an http.Handler that handles the routing for the vault piece API.
//
// It mounts the "/" route to the VaultPieceEncrypt method and the "/{rid}" route to the VaultPieceDecrypt method.
//
// Returns:
// - http.Handler: The router that handles the vault piece API routing.
func (s *Rest) VaultPieceRoute() http.Handler {
	router := chi.NewRouter()
	router.Put("/", s.VaultPieceEncrypt)
	router.Get("/{rid}", s.VaultPieceDecrypt)
	return router
}

// VaultPieceEncrypt handles the encryption of a vault piece.
//
// It takes in an http.ResponseWriter and an http.Request as parameters.
// The function retrieves the request ID from the context and logs it.
// It then retrieves the authorization token from the request headers and uses it to authenticate the user.
// If the authentication fails, an appropriate error response is returned.
// The function decodes the request body into a postgres.Piece struct.
// If the decoding fails, a bad request error response is returned.
// The function decodes the piece content from base64.
// If the decoding fails, a bad request error response is returned.
// The function retrieves the password from the request headers.
// If the password is missing, an internal server error response is returned.
// The function stores the piece in the database using the provided credentials.
// If the storage fails, an appropriate error response is returned.
// Finally, the function writes the response with the stored piece's ID and encodes it as JSON.
// If the encoding fails, an error message is logged.
func (s *Rest) VaultPieceEncrypt(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s VaultPieceEncryptHook", reqID)

	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(r.Context(), token)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var piece postgres.Piece
	if err := json.NewDecoder(r.Body).Decode(&piece); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var content = make([]byte, len(piece.Content))
	if _, err := base64.RawStdEncoding.Decode(content, ([]byte)(piece.Content)); err != nil {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rid, err := s.Store.StorePiece(r.Context(), piece, creds)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	var response struct {
		RID int64 `json:"rid"`
	}
	response.RID = (int64)(rid)
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		log.Printf("[ERROR] failed to write response: %s", err.Error())
	}
}

// VaultPieceDecrypt handles the decryption of a vault piece.
//
// It takes in an http.ResponseWriter and an http.Request as parameters.
// The function retrieves the request ID from the context and logs it.
// It then retrieves the authorization token from the request headers and uses it to authenticate the user.
// If the authentication fails, an appropriate error response is returned.
// The function retrieves the X-Password header from the request headers and assigns it to the creds.Passw field.
// If the password is missing, an unauthorized error response is returned.
// The function parses the "rid" URL parameter from the request and converts it to an integer.
// If the parsing fails, a bad request error response is returned.
// The function retrieves the vault piece with the specified resource ID from the database using the provided credentials.
// If the retrieval fails, an appropriate error response is returned.
// The function creates a response struct with the decrypted piece's metadata and encodes it as a base64-encoded string.
// The function writes the response with the appropriate status code and encodes it as JSON.
// If the encoding fails, an error message is logged.
func (s *Rest) VaultPieceDecrypt(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s VaultPieceDecryptHook", reqID)

	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(r.Context(), token)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	creds.Passw = r.Header.Get("X-Password")
	if creds.Passw == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	rid, err := strconv.Atoi(chi.URLParam(r, "rid"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	piece, err := s.Store.RestorePiece(r.Context(), (postgres.ResourceID)(rid), creds)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var response postgres.Piece
	response.Meta = piece.Meta
	response.Content = []byte(base64.RawStdEncoding.EncodeToString(
		bytes.ReplaceAll(
			piece.Content,
			[]byte{'\x00'},
			[]byte{},
		),
	))
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] failed to write response: %s", err.Error())
	}
}

// VaultBlobRoute returns an http.Handler that handles the routing for the vault blob API.
//
// It mounts the "/" route to the VaultBLobEncrypt method and the "/{rid}" route to the VaultBLobDecrypt method.
//
// Returns:
// - http.Handler: The router that handles the vault blob API routing.
func (s *Rest) VaultBlobRoute() http.Handler {
	router := chi.NewRouter()
	router.Put("/", s.VaultBLobEncrypt)
	router.Get("/{rid}", s.VaultBLobDecrypt)
	return router
}

// VaultBLobEncrypt handles the encryption of a blob using the provided credentials.
//
// It takes an http.ResponseWriter and an http.Request as parameters.
// The function retrieves the password from the request headers and checks if it is empty.
// If the password is empty, it returns an HTTP 401 Unauthorized response.
// It creates a postgres.Blob struct with the meta data from the request headers and the content from the request body.
// It calls the StoreBlob method of the Rest struct's Store field to store the blob and returns the resource ID.
// If an error occurs during the storage process, it checks if the error is postgres.ErrUserUnauthorized.
// If it is, it returns an HTTP 401 Unauthorized response. Otherwise, it returns an HTTP 500 Internal Server Error response.
// If the storage process is successful, it writes an HTTP 201 Created response to the http.ResponseWriter.
// It creates a response struct with the resource ID and encodes it to JSON.
// If an error occurs during the encoding process, it logs an error message.
func (s *Rest) VaultBLobEncrypt(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s VaultBlobEncryptHook", reqID)

	var creds postgres.Creds
	creds.Passw = r.Header.Get("X-Password")
	if creds.Passw == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	blob := postgres.Blob{
		Meta:    r.Header.Get("X-Meta"),
		Content: r.Body,
	}
	rid, err := s.Store.StoreBlob(r.Context(), blob, creds)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	var response struct {
		RID int64 `json:"rid"`
	}
	response.RID = (int64)(rid)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] Failed to write response: %s", err.Error())
	}
}

// VaultBLobDecrypt decrypts a blob from the vault.
//
// It takes an http.ResponseWriter and an http.Request as parameters.
// The function retrieves the password from the request headers and checks if it is empty.
// If the password is empty, it returns an HTTP 401 Unauthorized response.
// It retrieves the resource ID from the URL parameter "rid" and checks if it is valid.
// If the resource ID is invalid, it returns an HTTP 400 Bad Request response.
// It creates a postgres.Creds struct with the password from the request headers and calls the Identity method of the Rest struct's Store field to authenticate the user.
// If an error occurs during the authentication process, it checks if the error is postgres.ErrUserUnauthorized.
// If it is, it returns an HTTP 401 Unauthorized response. Otherwise, it returns an HTTP 500 Internal Server Error response.
// It calls the RestoreBlob method of the Rest struct's Store field to retrieve the blob and returns the decrypted content.
// If an error occurs during the retrieval process, it checks if the error is postgres.ErrUserUnauthorized.
// If it is, it returns an HTTP 401 Unauthorized response. Otherwise, it returns an HTTP 500 Internal Server Error response.
// It sets the appropriate headers in the http.ResponseWriter and writes the decrypted content.
// If an error occurs during the writing process, it logs an error message.
func (s *Rest) VaultBLobDecrypt(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetReqID(r.Context())
	log.Printf("[INFO] reqID %s VaultBlobDecryptHook", reqID)
	token := r.Header.Get("Authorization")
	creds, err := s.Store.Identity(r.Context(), token)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rid, err := strconv.Atoi(chi.URLParam(r, "rid"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	creds.Passw = r.Header.Get("X-Password")
	if creds.Passw == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	blob, err := s.Store.RestoreBlob(r.Context(), (postgres.ResourceID)(rid), creds)
	if err != nil {
		if errors.Is(err, postgres.ErrUserUnauthorized) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer blob.Content.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	w.Header().Set("X-Meta", blob.Meta)
	w.WriteHeader(http.StatusOK)

	output := bufio.NewWriter(w)
	if _, err := output.ReadFrom(blob.Content); err != nil {
		log.Printf("[ERROR] failed to write content: %s", err.Error())
	}
	if err := output.Flush(); err != nil {
		log.Printf("[ERROR] failed to flush content: %s", err.Error())
	}
}
