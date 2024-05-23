package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"

	store "github.com/stsg/gophkeeper/pkg/store"
)

func (s *Rest) Register(w http.ResponseWriter, r *http.Request) {
	// TODO: implement 	var requestBody map[string]any
	var rBody map[string]any
	if err := json.NewDecoder(r.Body).Decode(&rBody); err != nil {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	var user store.User

	if val, ok := rBody["username"].(string); ok {
		user.Login = val
	} else {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	if val, ok := rBody["password"].(string); ok {
		user.Passw = val
	} else {
		var status = http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}

	// rest.SendErrorJSON(w, r, log.Default(), http.StatusOK, fmt.Errorf("login %s password %s", login, password), "not yet implemented")

	rest.RenderJSON(w, &user)

}

func (s *Rest) Login(w http.ResponseWriter, r *http.Request) {
	// TODO: implement me
	rest.SendErrorJSON(w, r, log.Default(), http.StatusNotImplemented, fmt.Errorf("not yet implemented"), "not yet implemented")

}

func (s *Rest) Vault(w http.ResponseWriter, r *http.Request) {
	// TODO: implement me
	rest.SendErrorJSON(w, r, log.Default(), http.StatusNotImplemented, fmt.Errorf("not yet implemented"), "not yet implemented")

}
