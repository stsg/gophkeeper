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

// Decompress middleware
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

// Authorization middleware
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

// Authorization middleware
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
