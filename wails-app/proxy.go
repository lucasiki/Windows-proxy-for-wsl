package main

import (
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

// ConnectionEvent is emitted for each successfully proxied connection.
type ConnectionEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	SourceIP   string    `json:"sourceIP"`
	ListenPort int       `json:"listenPort"`
	WSLPort    int       `json:"wslPort"`
}

// ProxyState represents the current state of the engine.
type ProxyState int

const (
	StateStopped ProxyState = iota
	StateRunning
	StatePaused
)

// ProxyEngine orchestrates multiple per-port TCP proxy servers.
type ProxyEngine struct {
	OnConnection  func(ConnectionEvent)
	OnError       func(listenPort, wslPort int, msg string)
	OnStateChange func(state ProxyState, info *WSLInfo)

	servers []*proxyServer
	state   ProxyState
}

func NewProxyEngine(
	onConn func(ConnectionEvent),
	onErr func(int, int, string),
	onState func(ProxyState, *WSLInfo),
) *ProxyEngine {
	return &ProxyEngine{
		OnConnection:  onConn,
		OnError:       onErr,
		OnStateChange: onState,
	}
}

// Start detects WSL and begins listening on all configured port mappings.
func (e *ProxyEngine) Start(mappings []PortMapping) {
	info := DetectWSL()
	if info == nil {
		e.OnError(0, 0, "WSL não detectado. Certifique-se de que o WSL está rodando.")
		return
	}
	for _, m := range mappings {
		srv := newProxyServer(m.ListenPort, m.WSLPort, info.TargetIP,
			e.OnConnection,
			func(lp, wp int, msg string) { e.OnError(lp, wp, msg) },
		)
		if err := srv.start(); err != nil {
			e.OnError(m.ListenPort, m.WSLPort, err.Error())
			continue
		}
		e.servers = append(e.servers, srv)
	}
	e.state = StateRunning
	e.OnStateChange(StateRunning, info)
}

// Pause causes all servers to stop accepting new connections.
func (e *ProxyEngine) Pause() {
	if e.state != StateRunning {
		return
	}
	for _, s := range e.servers {
		s.setPaused(true)
	}
	e.state = StatePaused
	e.OnStateChange(StatePaused, nil)
}

// Resume re-enables connection acceptance.
func (e *ProxyEngine) Resume() {
	if e.state != StatePaused {
		return
	}
	for _, s := range e.servers {
		s.setPaused(false)
	}
	e.state = StateRunning
	e.OnStateChange(StateRunning, nil)
}

// Stop closes all listeners and resets state.
func (e *ProxyEngine) Stop() {
	if e.state == StateStopped {
		return
	}
	for _, s := range e.servers {
		s.stop()
	}
	e.servers = nil
	e.state = StateStopped
	e.OnStateChange(StateStopped, nil)
}

func (e *ProxyEngine) State() ProxyState { return e.state }

// ─────────────────────────────────────────────────────────────────────────────
// per-port server
// ─────────────────────────────────────────────────────────────────────────────

type proxyServer struct {
	listenPort int
	wslPort    int
	targetIP   string
	onConn     func(ConnectionEvent)
	onErr      func(int, int, string)
	listeners  []net.Listener
	paused     atomic.Bool
}

func newProxyServer(listenPort, wslPort int, targetIP string,
	onConn func(ConnectionEvent), onErr func(int, int, string),
) *proxyServer {
	return &proxyServer{
		listenPort: listenPort,
		wslPort:    wslPort,
		targetIP:   targetIP,
		onConn:     onConn,
		onErr:      onErr,
	}
}

func (s *proxyServer) start() error {
	port := strconv.Itoa(s.listenPort)

	ln4, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		return err
	}
	s.listeners = append(s.listeners, ln4)
	go s.acceptLoop(ln4)

	if ln6, err := net.Listen("tcp6", ":"+port); err == nil {
		s.listeners = append(s.listeners, ln6)
		go s.acceptLoop(ln6)
	}

	return nil
}

func (s *proxyServer) stop() {
	for _, ln := range s.listeners {
		_ = ln.Close()
	}
	s.listeners = nil
}

func (s *proxyServer) setPaused(v bool) { s.paused.Store(v) }

func (s *proxyServer) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		if s.paused.Load() {
			_ = conn.Close()
			continue
		}
		go s.handle(conn)
	}
}

func (s *proxyServer) handle(local net.Conn) {
	defer local.Close()

	srcIP, _, _ := net.SplitHostPort(local.RemoteAddr().String())

	remote, err := net.DialTimeout("tcp",
		net.JoinHostPort(s.targetIP, strconv.Itoa(s.wslPort)), 10*time.Second)
	if err != nil {
		s.onErr(s.listenPort, s.wslPort, err.Error())
		return
	}
	defer remote.Close()

	s.onConn(ConnectionEvent{
		Timestamp:  time.Now(),
		SourceIP:   srcIP,
		ListenPort: s.listenPort,
		WSLPort:    s.wslPort,
	})

	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(remote, local); done <- struct{}{} }()
	go func() { _, _ = io.Copy(local, remote); done <- struct{}{} }()
	<-done
}
