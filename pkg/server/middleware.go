package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	postgres "github.com/stsg/gophkeeper/pkg/store"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// LoggerFlag type
type LoggerFlag int

// logger flags enum
const (
	LogAll LoggerFlag = iota
	LogBody
)

const maxBody = 1024

type ContextKey string

const UserContextKey ContextKey = "user"

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Logger returns a middleware function that logs HTTP requests.
//
// It takes a logger (l) and a variadic parameter of LoggerFlags. The LoggerFlags
// determine what parts of the request to log. The function returns an http.Handler
// that wraps the provided http.Handler.
//
// The middleware function logs the following information:
// - HTTP method
// - URL
// - Remote address
// - HTTP status code
// - Bytes written
// - Time taken
// - Request body (if LogBody flag is set)
//
// The log message is formatted as follows:
// "[INFO] REST {HTTP method} - {URL} - {Remote address} - {HTTP status code} ({Bytes written}) - {Time taken} {Request body}"
// Logger middleware prints http log. Customized by set of LoggerFlag
func Logger(l log.L, flags ...LoggerFlag) func(http.Handler) http.Handler {

	inFlags := func(f LoggerFlag) bool {
		for _, flg := range flags {
			if flg == LogAll || flg == f {
				return true
			}
		}
		return false
	}

	f := func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, 1)

			body := func() (result string) {
				if inFlags(LogBody) {
					if content, err := io.ReadAll(r.Body); err == nil {
						result = string(content)
						r.Body = io.NopCloser(bytes.NewReader(content))

						if len(result) > 0 {
							result = strings.Replace(result, "\n", " ", -1)
							result = reMultWhtsp.ReplaceAllString(result, " ")
						}

						if len(result) > maxBody {
							result = result[:maxBody] + "..."
						}
					}
				}
				return result
			}()

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				q := r.URL.String()
				if qun, err := url.QueryUnescape(q); err == nil {
					q = qun
				}
				l.Logf("[INFO] REST %s - %s - %s - %d (%d) - %v %s",
					r.Method, q, strings.Split(r.RemoteAddr, ":")[0],
					ww.Status(), ww.BytesWritten(), t2.Sub(t1), body)
			}()

			h.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}

	return f
}

// Decompress is a middleware function that decompresses the request body if it is gzip-encoded.
//
// It takes an http.Handler as input and returns an http.Handler.
// The returned http.Handler is responsible for handling the decompressed request.
//
// If the request body is gzip-encoded, it reads the compressed data from the request body,
// decompresses it, and replaces the original request body with the decompressed data.
// It also sets the "Content-Length" header to the length of the decompressed data.
//
// If there is an error during decompression, it sets the response status to 400 (Bad Request)
// and writes the error message to the response body.
//
// If the request body is not gzip-encoded, it simply passes the request to the next handler in the chain.
//
// The function logs a message indicating that the Decompress middleware is enabled.
func Decompress() func(http.Handler) http.Handler {

	log.Print("Decompress middleware enabled")

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Encoding") == "gzip" {
				reader, err := gzip.NewReader(r.Body)
				if err != nil {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusBadRequest)
					_, err := w.Write([]byte(err.Error()))
					if err != nil {
						return
					}
					return
				}
				defer reader.Close()

				buf := new(strings.Builder)
				_, err = io.Copy(buf, reader)
				if err != nil {
					w.Header().Set("Content-Type", "text/plain")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(err.Error()))
					return
				}
				r.Body = io.NopCloser(strings.NewReader(buf.String()))
				r.Header.Set("Content-Length", string(rune(len(buf.String()))))
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}

	return f
}

// Authorization returns a middleware function that checks for authorization.
//
// The middleware function takes an http.Handler as input and returns an http.Handler.
// The returned http.Handler checks if the "X-Password" header is present in the request.
// If the header is missing, it returns a 401 Unauthorized status code.
// If the header is present, it calls the next handler in the chain.
//
// Parameters:
// - h: The http.Handler to be wrapped by the middleware.
//
// Returns:
// - An http.Handler that performs authorization checks.
func Authorization() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Password") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}

// Authorization candidate to middleware
// TODO -
func AuthRequired(s *Rest) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			creds, err := s.Store.Identity(r.Context(), token)
			if err != nil {
				if errors.Is(err, postgres.ErrUserUnauthorized) {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			r.Header.Set("X-Password", creds.Passw)
			h.ServeHTTP(w, r)

		})
	}
}
