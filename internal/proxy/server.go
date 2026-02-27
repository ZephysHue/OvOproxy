package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"zephy/internal/config"
	"zephy/internal/hosts"
)

type Server struct {
	profile config.Profile
	hosts   *hosts.Table
	client  *http.Client
}

func New(profile config.Profile, table *hosts.Table) *Server {
	tr := &http.Transport{
		Proxy:               nil,
		MaxIdleConns:        100,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &Server{
		profile: profile,
		hosts:   table,
		client:  &http.Client{Transport: tr},
	}
}

func (s *Server) ListenAndServe() error {
	return s.ListenAndServeWithContext(context.Background())
}

func (s *Server) ListenAndServeWithContext(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.profile.ListenIP, s.profile.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      s,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Printf("profile=%s listening on %s", s.profile.Name, addr)

	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		s.handleConnect(w, r)
		return
	}
	s.handleHTTP(w, r)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	targetHost, targetPort, err := net.SplitHostPort(r.Host)
	if err != nil {
		http.Error(w, "bad CONNECT host", http.StatusBadRequest)
		return
	}

	dialHost := targetHost
	if mapped, ok := s.hosts.Resolve(targetHost); ok {
		dialHost = mapped
	}
	target := net.JoinHostPort(dialHost, targetPort)

	dst, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		dst.Close()
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}

	src, _, err := hj.Hijack()
	if err != nil {
		dst.Close()
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	_, _ = src.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go transfer(dst, src)
	go transfer(src, dst)
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""

	host := outReq.URL.Hostname()
	if host == "" {
		host = outReq.Host
	}

	if mapped, ok := s.hosts.Resolve(host); ok {
		port := outReq.URL.Port()
		if port == "" {
			if strings.EqualFold(outReq.URL.Scheme, "https") {
				port = "443"
			} else {
				port = "80"
			}
		}
		outReq.URL.Host = net.JoinHostPort(mapped, port)
	}

	resp, err := s.client.Do(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	_, _ = io.Copy(dst, src)
}
