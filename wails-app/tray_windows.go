//go:build windows

package main

import (
	_ "embed"
	"context"
	"encoding/binary"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.png
var trayIcon []byte

// wrapPNGInICO wraps a PNG file inside a minimal ICO container.
// Windows Vista+ LoadImage can handle ICO files that embed PNG data.
func wrapPNGInICO(png []byte) []byte {
	var w, h byte
	if len(png) >= 24 {
		width := binary.BigEndian.Uint32(png[16:20])
		height := binary.BigEndian.Uint32(png[20:24])
		if width < 256 {
			w = byte(width)
		}
		if height < 256 {
			h = byte(height)
		}
	}
	pngSize := uint32(len(png))
	const headerSize = uint32(6 + 16)

	buf := make([]byte, 0, int(headerSize)+len(png))
	// ICONDIR
	buf = append(buf, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00)
	// ICONDIRENTRY
	buf = append(buf, w, h, 0x00, 0x00, 0x01, 0x00, 0x20, 0x00)
	buf = binary.LittleEndian.AppendUint32(buf, pngSize)
	buf = binary.LittleEndian.AppendUint32(buf, headerSize)
	buf = append(buf, png...)
	return buf
}

func startTray(ctx context.Context) {
	systray.Run(func() {
		systray.SetIcon(wrapPNGInICO(trayIcon))
		systray.SetTooltip("WSL Proxy")

		mShow := systray.AddMenuItem("Mostrar", "Mostrar janela")
		mQuit := systray.AddMenuItem("Fechar", "Fechar aplicação")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					runtime.WindowShow(ctx)
				case <-mQuit.ClickedCh:
					runtime.Quit(ctx)
				}
			}
		}()
	}, func() {})
}
