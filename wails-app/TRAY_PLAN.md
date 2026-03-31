# Plan: Frameless window + system tray (minimize to tray / close)

## Context

Remove the native Windows title bar and replace with a custom React title bar.
- **Botão ─ (minimizar)** → esconde a janela e mantém ícone na system tray
- **Botão ✕ (fechar)** → encerra o processo completamente
- Clique no ícone da tray → restaura a janela

---

## Files to modify

| File | Change |
|---|---|
| `wails-app/main.go` | Add `Frameless: true` to Wails options |
| `wails-app/app.go` | Call `startTray(ctx)` inside `startup()` |
| NEW `wails-app/tray.go` | Systray setup (icon, show/quit menu items) |
| `wails-app/go.mod` | Add `github.com/getlantern/systray v1.2.2` dependency |
| `wails-app/frontend/src/components/Header.tsx` | Draggable header + close/minimize buttons |

---

## Backend changes

### `main.go` — add `Frameless: true`
```go
err = wails.Run(&options.App{
    ...
    Frameless: true,   // ← add this
    ...
})
```

### NEW `tray.go`
```go
package main

import (
    _ "embed"
    "context"

    "github.com/getlantern/systray"
    "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.png
var trayIcon []byte

func startTray(ctx context.Context) {
    start, _ := systray.RunWithExternalLoop(func() {
        systray.SetIcon(trayIcon)
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
    start()
}
```

### `app.go` — call `startTray` no final de `startup()`
```go
func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
    a.engine = NewProxyEngine(...)
    go startTray(ctx)   // ← add this line
}
```

### `go.mod` — add dependency
```
github.com/getlantern/systray v1.2.2
```
Rodar `go mod tidy` localmente ou deixar o Docker resolver.

---

## Frontend changes

### `Header.tsx`

1. Adicionar `style={{ ['--wails-draggable' as string]: 'drag' }}` no div raiz do header — permite arrastar a janela pela barra
2. Importar `Quit` e `WindowHide` do runtime:
   ```ts
   import { Quit, WindowHide } from '../wailsjs/runtime/runtime'
   ```
3. Adicionar dois botões no canto direito (depois do status dot):
   - **─** → `WindowHide()` (minimiza para tray)
   - **✕** → `Quit()` (fecha o processo)
4. Botões com `style={{ ['--wails-draggable' as string]: 'no-drag' }}` para não capturar o drag

Exemplo dos botões:
```tsx
const btnStyle: React.CSSProperties = {
  background: 'transparent',
  color: 'var(--text-muted)',
  padding: '2px 8px',
  border: 'none',
  fontSize: 14,
  lineHeight: 1,
  borderRadius: 3,
  ['--wails-draggable' as string]: 'no-drag',
}

<button style={btnStyle} onClick={WindowHide} title="Minimizar para tray">─</button>
<button style={{ ...btnStyle, color: 'var(--red)' }} onClick={Quit} title="Fechar">✕</button>
```

---

## Verification

1. `wails dev` no Windows → janela abre sem barra nativa
2. Header é arrastável (mover a janela pelo título)
3. Botão **✕** → processo encerra completamente
4. Botão **─** → janela some, ícone aparece na system tray
5. Clicar no ícone da tray → "Mostrar" restaura a janela
6. "Fechar" na tray → encerra o processo
