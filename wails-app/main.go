package main

import (
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Write logs next to the exe so we can diagnose startup failures.
	if f, err := os.OpenFile("wsl_proxy_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		log.SetOutput(f)
		defer f.Close()
	}

	app := NewApp()

	// fs.Sub is required: //go:embed stores files under "frontend/dist/…",
	// but the Wails asset server expects index.html at the FS root.
	subAssets, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatalf("fs.Sub failed: %v", err)
	}

	err = wails.Run(&options.App{
		Title:            "WSL Proxy",
		Frameless:        true,
		Width:            700,
		Height:           580,
		MinWidth:         600,
		MinHeight:        480,
		AssetServer:      &assetserver.Options{Assets: subAssets},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 46, A: 1},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})
	if err != nil {
		log.Fatalf("wails.Run error: %v", err)
	}
}
