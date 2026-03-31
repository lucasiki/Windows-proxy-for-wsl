package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var portRe = regexp.MustCompile(`^(\d+)(?::(\d+))?$`)

// App is the Wails application struct. Its exported methods are callable from
// the React frontend via the generated wailsjs bindings.
type App struct {
	ctx      context.Context
	engine   *ProxyEngine
	mappings []PortMapping
	mu       sync.Mutex
	connCount int
	wslInfo  *WSLInfo
}

func NewApp() *App {
	a := &App{}
	a.mappings = loadSettings()
	return a
}

// startup is called by Wails after the window is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.engine = NewProxyEngine(
		func(e ConnectionEvent) {
			a.mu.Lock()
			a.connCount++
			count := a.connCount
			a.mu.Unlock()

			runtime.EventsEmit(ctx, "proxy:connection", map[string]interface{}{
				"time":       e.Timestamp.Format("15:04:05"),
				"sourceIP":   e.SourceIP,
				"listenPort": e.ListenPort,
				"wslPort":    e.WSLPort,
			})
			runtime.EventsEmit(ctx, "proxy:connCount", map[string]interface{}{
				"count": count,
			})
		},
		func(lp, wp int, msg string) {
			portInfo := ""
			if lp > 0 {
				portInfo = fmt.Sprintf(" (porta %d)", lp)
			}
			runtime.EventsEmit(ctx, "proxy:error", map[string]interface{}{
				"message": fmt.Sprintf("Erro%s: %s", portInfo, msg),
			})
		},
		func(state ProxyState, info *WSLInfo) {
			if info != nil {
				a.mu.Lock()
				a.wslInfo = info
				a.mu.Unlock()
			}
			stateStr := stateToString(state)
			payload := map[string]interface{}{"state": stateStr}
			if info != nil {
				payload["wslInfo"] = info
			}
			runtime.EventsEmit(ctx, "proxy:state", payload)
		},
	)
	go startTray(ctx) 
}

// ─────────────────────────────────────────────────────────────────────────────
// Methods exposed to the frontend
// ─────────────────────────────────────────────────────────────────────────────

// GetWSLInfo detects the current WSL network configuration.
func (a *App) GetWSLInfo() *WSLInfo {
	return DetectWSL()
}

// GetMappings returns the current list of port mappings.
func (a *App) GetMappings() []PortMapping {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mappings == nil {
		return []PortMapping{}
	}
	return a.mappings
}

// GetState returns the current proxy state as a string.
func (a *App) GetState() string {
	if a.engine == nil {
		return "stopped"
	}
	return stateToString(a.engine.State())
}

// AddMapping parses a port spec ("3000" or "3000:3001") and adds it.
func (a *App) AddMapping(spec string) error {
	lp, wp, ok := parsePort(spec)
	if !ok {
		return fmt.Errorf(`formato inválido — use "3000" ou "3000:3001"`)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, m := range a.mappings {
		if m.ListenPort == lp {
			return fmt.Errorf("porta %d já foi adicionada", lp)
		}
	}
	a.mappings = append(a.mappings, PortMapping{
		ID:         newUUID(),
		ListenPort: lp,
		WSLPort:    wp,
	})
	saveSettings(a.mappings)
	return nil
}

// RemoveMapping removes a mapping by its UUID.
func (a *App) RemoveMapping(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, m := range a.mappings {
		if m.ID == id {
			a.mappings = append(a.mappings[:i], a.mappings[i+1:]...)
			saveSettings(a.mappings)
			return nil
		}
	}
	return fmt.Errorf("mapeamento não encontrado")
}

// Start starts the proxy engine.
func (a *App) Start() error {
	if a.engine == nil {
		return fmt.Errorf("aplicação ainda não está pronta")
	}
	a.mu.Lock()
	mappings := make([]PortMapping, len(a.mappings))
	copy(mappings, a.mappings)
	a.mu.Unlock()

	if len(mappings) == 0 {
		return fmt.Errorf("adicione pelo menos uma porta antes de iniciar")
	}
	if a.engine.State() == StatePaused {
		a.engine.Resume()
		return nil
	}
	go a.engine.Start(mappings)
	return nil
}

// Pause pauses or resumes the proxy engine.
func (a *App) Pause() {
	if a.engine == nil {
		return
	}
	switch a.engine.State() {
	case StateRunning:
		a.engine.Pause()
	case StatePaused:
		a.engine.Resume()
	}
}

// Stop stops the proxy engine.
func (a *App) Stop() {
	if a.engine == nil {
		return
	}
	a.engine.Stop()
}

// ClearLog resets the connection counter (log entries live in the frontend).
func (a *App) ClearLog() {
	a.mu.Lock()
	a.connCount = 0
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "proxy:connCount", map[string]interface{}{"count": 0})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func stateToString(s ProxyState) string {
	switch s {
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	default:
		return "stopped"
	}
}

func parsePort(raw string) (listenPort, wslPort int, ok bool) {
	m := portRe.FindStringSubmatch(raw)
	if m == nil {
		return
	}
	lp, err := strconv.Atoi(m[1])
	if err != nil || lp < 1 || lp > 65535 {
		return
	}
	wp := lp
	if m[2] != "" {
		wp, err = strconv.Atoi(m[2])
		if err != nil || wp < 1 || wp > 65535 {
			return
		}
	}
	return lp, wp, true
}
