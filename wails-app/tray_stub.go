//go:build !windows

package main

import "context"

func startTray(ctx context.Context) {
    // No-op no Linux
}
