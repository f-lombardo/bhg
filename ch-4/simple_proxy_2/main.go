package main

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return u
}

func newProxy(target *url.URL, originalHost string) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Cambia destinazione: scheme, host e path.
			pr.SetURL(target)

			// Il backend deve vedere l'Host originale
			// inviato dal client.
			pr.Out.Host = originalHost

			// Aggiunge:
			// X-Forwarded-For
			// X-Forwarded-Host
			// X-Forwarded-Proto
			pr.SetXForwarded()
		},

		ErrorHandler: func(
			w http.ResponseWriter,
			r *http.Request,
			err error,
		) {
			log.Printf(
				"proxy error: host=%q method=%s path=%q: %v",
				r.Host,
				r.Method,
				r.URL.RequestURI(),
				err,
			)

			http.Error(
				w,
				"backend unavailable",
				http.StatusBadGateway,
			)
		},
	}
}

func main() {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		// Host contiene il valore dell'header Host.
		//
		// Esempio:
		//   site-a.example
		//   site-a.example:8000
		originalHost := r.Host

		// Registra ogni sito richiesto.
		log.Printf(
			"request: host=%q method=%s path=%q client=%q",
			originalHost,
			r.Method,
			r.URL.RequestURI(),
			r.RemoteAddr,
		)

		// Crea il reverse proxy per il backend
		// associato all'host richiesto.
		proxy := newProxy(mustParseURL(r.RequestURI), originalHost)

		proxy.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              ":8123",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("reverse proxy listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
