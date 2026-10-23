package dnsengine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// ServerConfig configures the DNS server listeners.
type ServerConfig struct {
	ListenAddr       string
	Port             int
	Handler          dns.Handler
	AllowNonLoopback bool
}

// Server manages dual UDP and TCP DNS listeners bound to loopback.
type Server struct {
	mu               sync.RWMutex
	listenAddr       string
	configuredPort   int
	boundAddr        string
	handler          dns.Handler
	allowNonLoopback bool

	udpServer   *dns.Server
	tcpServer   *dns.Server
	udpConn     net.PacketConn
	tcpListener net.Listener
	running     bool
	wg          sync.WaitGroup
}

// NewServer creates a new DNS server with safe loopback defaults.
func NewServer(cfg ServerConfig) *Server {
	addr := strings.TrimSpace(cfg.ListenAddr)
	if addr == "" {
		addr = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 && cfg.ListenAddr == "" {
		port = 5353
	}

	return &Server{
		listenAddr:       addr,
		configuredPort:   port,
		handler:          cfg.Handler,
		allowNonLoopback: cfg.AllowNonLoopback,
	}
}

// isLoopbackAddress checks whether the given host is a loopback interface.
func isLoopbackAddress(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// Start launches the UDP and TCP listeners concurrently.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("dns server is already running")
	}

	// Safety check: prevent binding non-loopback unless explicitly configured
	if !s.allowNonLoopback && !isLoopbackAddress(s.listenAddr) {
		return fmt.Errorf("binding to non-loopback address %q is disabled by default for security", s.listenAddr)
	}

	// 1. Bind TCP listener first to acquire port (including port 0 dynamic assignment)
	bindTCPTarget := net.JoinHostPort(s.listenAddr, strconv.Itoa(s.configuredPort))
	tcpListener, err := net.Listen("tcp", bindTCPTarget)
	if err != nil {
		return fmt.Errorf("failed to bind TCP listener on %s: %w", bindTCPTarget, err)
	}
	s.tcpListener = tcpListener

	tcpAddr, ok := tcpListener.Addr().(*net.TCPAddr)
	if !ok {
		_ = tcpListener.Close()
		return errors.New("failed to retrieve TCP listener address")
	}
	actualPort := tcpAddr.Port

	// 2. Bind UDP PacketConn to the exact same port
	bindUDPTarget := net.JoinHostPort(s.listenAddr, strconv.Itoa(actualPort))
	udpConn, err := net.ListenPacket("udp", bindUDPTarget)
	if err != nil {
		_ = tcpListener.Close()
		return fmt.Errorf("failed to bind UDP listener on %s: %w", bindUDPTarget, err)
	}
	s.udpConn = udpConn

	s.boundAddr = net.JoinHostPort(s.listenAddr, strconv.Itoa(actualPort))

	// 3. Create miekg/dns server instances using the bound sockets
	s.tcpServer = &dns.Server{
		Listener: s.tcpListener,
		Handler:  s.handler,
	}
	s.udpServer = &dns.Server{
		PacketConn: s.udpConn,
		Handler:    s.handler,
	}

	s.running = true

	// Launch TCP server in background goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		_ = s.tcpServer.ActivateAndServe()
	}()

	// Launch UDP server in background goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		_ = s.udpServer.ActivateAndServe()
	}()

	return nil
}

// Shutdown gracefully shuts down both UDP and TCP listeners without socket leaks.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	tcpSrv := s.tcpServer
	udpSrv := s.udpServer
	s.mu.Unlock()

	var errs []error
	if udpSrv != nil {
		if err := udpSrv.ShutdownContext(ctx); err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, err)
		}
	}
	if tcpSrv != nil {
		if err := tcpSrv.ShutdownContext(ctx); err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, err)
		}
	}

	// Wait for worker goroutines to terminate
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		errs = append(errs, ctx.Err())
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during server shutdown: %v", errs)
	}
	return nil
}

// IsRunning reports whether the server is currently serving queries.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Addr returns the bound ip:port string, or empty string if not running.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.boundAddr
}

// ListenAddr returns the configured listen address.
func (s *Server) ListenAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listenAddr
}

// Port returns the configured listen port.
func (s *Server) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configuredPort
}
