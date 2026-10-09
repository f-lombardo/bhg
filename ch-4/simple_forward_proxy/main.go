package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	forwardedHeaderName  = "X-Proxy-Custom"
	forwardedHeaderValue = "simple-forward-proxy"
)

func main() {
	transport := http.DefaultTransport

	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		log.Printf(
			"request: host=%q method=%s path=%q client=%q",
			r.Host,
			r.Method,
			r.URL.RequestURI(),
			r.RemoteAddr,
		)

		if r.Method == http.MethodConnect {
			handleHTTPSConnect(w, r)
			return
		}

		forwardHTTP(w, r, transport)
	})

	server := &http.Server{
		Addr:              ":8123",
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("reverse proxy listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func forwardHTTP(
	w http.ResponseWriter,
	r *http.Request,
	transport http.RoundTripper,
) {
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""

	if outReq.URL.Scheme == "" {
		outReq.URL.Scheme = "http"
	}
	if outReq.URL.Host == "" {
		outReq.URL.Host = r.Host
	}
	outReq.Header.Set(forwardedHeaderName, forwardedHeaderValue)

	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		log.Printf(
			"proxy error: host=%q method=%s path=%q: %v",
			r.Host,
			r.Method,
			r.URL.RequestURI(),
			err,
		)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf(
			"response copy error: host=%q method=%s path=%q: %v",
			r.Host,
			r.Method,
			r.URL.RequestURI(),
			err,
		)
	}
}

func handleHTTPSConnect(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("hijack error: host=%q: %v", r.Host, err)
		return
	}

	targetConn, err := net.DialTimeout("tcp", withDefaultPort(r.Host, "443"), 10*time.Second)
	if err != nil {
		log.Printf("connect error: host=%q: %v", r.Host, err)
		_, _ = io.WriteString(clientConn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		_ = clientConn.Close()
		return
	}

	_, _ = io.WriteString(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")

	done := make(chan struct{}, 2)
	go tunnelCopy(done, targetConn, clientConn)
	go tunnelCopy(done, clientConn, targetConn)

	<-done
	_ = clientConn.Close()
	_ = targetConn.Close()
	<-done
}

func tunnelCopy(done chan<- struct{}, dst net.Conn, src net.Conn) {
	_, _ = io.Copy(dst, src)
	done <- struct{}{}
}

func copyHeaders(dst, src http.Header) {
	for k, vals := range src {
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

func withDefaultPort(host, fallbackPort string) string {
	if strings.Contains(host, ":") {
		return host
	}
	return net.JoinHostPort(host, fallbackPort)
}
