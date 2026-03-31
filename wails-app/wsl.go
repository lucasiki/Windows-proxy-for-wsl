package main

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"
)

// WSLInfo holds the detected WSL network information.
type WSLInfo struct {
	IP       string `json:"ip"`
	Mode     string `json:"mode"`
	TargetIP string `json:"targetIP"`
}

// DetectWSL runs "wsl hostname -I" and returns WSL network info.
// Returns nil if WSL is not available or not running.
func DetectWSL() *WSLInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "wsl", "hostname", "-I").Output()
	if err != nil {
		return nil
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) == 0 {
		return nil
	}
	wslIP := parts[0]

	localIPs := localHostIPs()
	for _, ip := range localIPs {
		if ip == wslIP {
			return &WSLInfo{IP: wslIP, Mode: "mirrored", TargetIP: "127.0.0.1"}
		}
	}
	return &WSLInfo{IP: wslIP, Mode: "NAT", TargetIP: wslIP}
}

func localHostIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var ips []string
	for _, a := range addrs {
		switch v := a.(type) {
		case *net.IPNet:
			if ip4 := v.IP.To4(); ip4 != nil {
				ips = append(ips, ip4.String())
			}
		case *net.IPAddr:
			if ip4 := v.IP.To4(); ip4 != nil {
				ips = append(ips, ip4.String())
			}
		}
	}
	return ips
}
