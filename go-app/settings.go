package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PortMapping represents a listen_port → wsl_port pair.
type PortMapping struct {
	ID         string `json:"id"`
	ListenPort int    `json:"listen_port"`
	WSLPort    int    `json:"wsl_port"`
}

type settings struct {
	PortMappings []PortMapping `json:"port_mappings"`
}

func settingsPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "wsl_proxy_settings.json"
	}
	return filepath.Join(filepath.Dir(exe), "wsl_proxy_settings.json")
}

func loadSettings() []PortMapping {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return nil
	}
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	// Ensure every mapping has an ID.
	for i := range s.PortMappings {
		if s.PortMappings[i].ID == "" {
			s.PortMappings[i].ID = newUUID()
		}
	}
	return s.PortMappings
}

func saveSettings(mappings []PortMapping) {
	data, err := json.MarshalIndent(settings{PortMappings: mappings}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(settingsPath(), data, 0644)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
