package main

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

//go:embed assets/icon.png
var iconBytes []byte

// dispatch schedules fn to run on the Fyne main goroutine.
// Uses a per-app channel + ticker since fyne.Do is only available in Fyne ≥ 2.6.
var mainQueue = make(chan func(), 256)

func dispatch(fn func()) { mainQueue <- fn }

func startDispatcher() {
	go func() {
		t := time.NewTicker(16 * time.Millisecond) // ~60 fps drain
		defer t.Stop()
		for range t.C {
			for {
				select {
				case fn := <-mainQueue:
					fn()
				default:
					goto done
				}
			}
		done:
		}
	}()
}

var portRe = regexp.MustCompile(`^(\d+)(?::(\d+))?$`)

func main() {
	a := app.New()
	a.Settings().SetTheme(&darkTheme{})

	icon := fyne.NewStaticResource("icon.png", iconBytes)
	a.SetIcon(icon)

	w := a.NewWindow("WSL Proxy")
	w.Resize(fyne.NewSize(640, 560))
	w.SetIcon(icon)

	// System tray — only available on desktop builds (always true on Windows).
	// Close (X) hides to tray; "Fechar" in tray menu actually quits.
	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(icon)
		desk.SetSystemTrayMenu(fyne.NewMenu("WSL Proxy",
			fyne.NewMenuItem("Mostrar", func() { w.Show() }),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Fechar", func() { a.Quit() }),
		))
		w.SetCloseIntercept(func() { w.Hide() })
	}

	startDispatcher()
	ui := newAppUI(w)
	w.SetContent(ui.root())
	w.ShowAndRun()
}

// ─────────────────────────────────────────────────────────────────────────────
// appUI
// ─────────────────────────────────────────────────────────────────────────────

type appUI struct {
	win         fyne.Window
	engine      *ProxyEngine
	mappings    []PortMapping
	lastWSLInfo *WSLInfo

	// header
	wslInfoLabel *widget.Label
	statusLabel  *canvas.Text // shows "● PARADO" etc. with exact palette color

	// port panel
	portEntry  *widget.Entry
	btnAdd     *widget.Button
	mappingBox *fyne.Container // VBox, rebuilt on change
	btnStart   *widget.Button
	//btnPause   *widget.Button
	btnStop    *widget.Button

	// log panel
	logEntries  []logEntry
	logList     *widget.List
	logCountLbl *widget.Label
}

type logEntry struct {
	text    string
	isError bool
}

func newAppUI(win fyne.Window) *appUI {
	ui := &appUI{win: win}
	ui.mappings = loadSettings()

	ui.engine = NewProxyEngine(
		func(e ConnectionEvent) {
			dispatch(func() { ui.appendLog(e) })
		},
		func(lp, wp int, msg string) {
			dispatch(func() { ui.appendError(lp, wp, msg) })
		},
		func(state ProxyState, info *WSLInfo) {
			dispatch(func() { ui.applyState(state, info) })
		},
	)
	return ui
}

// ─────────────────────────────────────────────────────────────────────────────
// Root layout: header (fixed) + padded content (expands)
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) root() fyne.CanvasObject {
	header := ui.buildHeader()
	portPanel := ui.buildPortPanel()
	logPanel := ui.buildLogPanel()

	// Gap between panels
	gap := canvas.NewRectangle(colorBGPrimary)
	gap.SetMinSize(fyne.NewSize(0, 14))

	// portPanel sits at top (fixed height), logPanel expands into remaining space.
	inner := container.NewBorder(
		container.NewVBox(portPanel, gap),
		nil, nil, nil,
		logPanel,
	)

	return container.NewBorder(
		header, nil, nil, nil,
		container.NewPadded(inner),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Header
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) buildHeader() fyne.CanvasObject {
	title := widget.NewLabel("WSL Proxy")
	title.TextStyle = fyne.TextStyle{Bold: true}

	ui.wslInfoLabel = widget.NewLabel("WSL não detectado")
	ui.wslInfoLabel.Importance = widget.LowImportance

	ui.statusLabel = canvas.NewText("● PARADO", colorRed)
	ui.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	ui.statusLabel.TextSize = 13

	right := container.NewHBox(ui.wslInfoLabel, ui.statusLabel)
	bg := canvas.NewRectangle(colorBGSecondary)
	return container.NewStack(bg,
		container.NewPadded(container.NewBorder(nil, nil, title, right)),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Port mapping panel
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) buildPortPanel() fyne.CanvasObject {
	titleLbl := widget.NewLabel("MAPEAMENTO DE PORTAS")
	titleLbl.TextStyle = fyne.TextStyle{Bold: true}
	titleLbl.Importance = widget.LowImportance
	titleRow := container.NewStack(
		canvas.NewRectangle(colorBGSurface),
		container.NewPadded(titleLbl),
	)

	// Input row
	ui.portEntry = widget.NewEntry()
	ui.portEntry.SetPlaceHolder("ex: 3000  ou  3000:3001  (porta Windows:porta WSL)")
	ui.portEntry.OnSubmitted = func(_ string) { ui.addMapping() }

	ui.btnAdd = widget.NewButton("+ Adicionar", ui.addMapping)
	ui.btnAdd.Importance = widget.HighImportance
	inputRow := container.NewBorder(nil, nil, nil, ui.btnAdd, ui.portEntry)

	// Mapping box: VBox rebuilt on every change — no type assertions needed.
	ui.mappingBox = container.NewVBox()
	ui.rebuildMappings()
	scroll := container.NewVScroll(ui.mappingBox)
	scroll.SetMinSize(fyne.NewSize(0, 100))

	// Controls row
	ui.btnStart = widget.NewButton("▶ Iniciar", ui.onStart)
	ui.btnStart.Importance = widget.HighImportance

	//ui.btnPause = widget.NewButton("⏸ Pausar", ui.onPause)
	//ui.btnPause.Hide()

	ui.btnStop = widget.NewButton("■ Parar", ui.onStop)
	ui.btnStop.Importance = widget.DangerImportance
	ui.btnStop.Disable()

	ctrlRow := container.NewHBox(ui.btnStart, ui.btnStop)
	controlsRow := container.NewStack(
		canvas.NewRectangle(colorBGSurface),
		container.NewPadded(ctrlRow),
	)

	bg := canvas.NewRectangle(colorBGSecondary)
	return container.NewStack(bg, container.NewVBox(
		titleRow,
		container.NewPadded(inputRow),
		scroll,
		controlsRow,
	))
}

// rebuildMappings clears and repopulates mappingBox. Must run on main goroutine.
func (ui *appUI) rebuildMappings() {
	locked := ui.engine.State() != StateStopped
	var objs []fyne.CanvasObject

	if len(ui.mappings) == 0 {
		ph := widget.NewLabel("Nenhuma porta configurada — adicione uma acima")
		ph.Importance = widget.LowImportance
		ph.Alignment = fyne.TextAlignCenter
		objs = []fyne.CanvasObject{container.NewCenter(ph)}
	} else {
		for _, m := range ui.mappings {
			m := m // capture loop var
			portLbl := widget.NewLabel(fmt.Sprintf(":%d → :%d", m.ListenPort, m.WSLPort))
			portLbl.TextStyle = fyne.TextStyle{Monospace: true}

			dirLbl := widget.NewLabel("Windows → WSL")
			dirLbl.Importance = widget.LowImportance

			removeBtn := widget.NewButton("✕", func() { ui.removeMapping(m.ID) })
			removeBtn.Importance = widget.DangerImportance
			if locked {
				removeBtn.Disable()
			}

			objs = append(objs, container.NewBorder(nil, nil, portLbl, removeBtn, dirLbl))
		}
	}

	ui.mappingBox.Objects = objs
	ui.mappingBox.Refresh()
}

// ─────────────────────────────────────────────────────────────────────────────
// Log panel
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) buildLogPanel() fyne.CanvasObject {
	titleLbl := widget.NewLabel("LOG DE CONEXÕES")
	titleLbl.TextStyle = fyne.TextStyle{Bold: true}
	titleLbl.Importance = widget.LowImportance

	ui.logCountLbl = widget.NewLabel("")
	ui.logCountLbl.Importance = widget.LowImportance

	btnClear := widget.NewButton("Limpar", func() {
		ui.logEntries = nil
		ui.logList.Refresh()
		ui.logCountLbl.SetText("")
	})

	titleRow := container.NewStack(
		canvas.NewRectangle(colorBGSurface),
		container.NewBorder(nil, nil,
			container.NewPadded(titleLbl),
			container.NewPadded(container.NewHBox(ui.logCountLbl, btnClear)),
		),
	)

	// Each log row is a single monospace Label — no type assertion risk.
	ui.logList = widget.NewList(
		func() int { return len(ui.logEntries) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(ui.logEntries) {
				return
			}
			e := ui.logEntries[id]
			lbl := obj.(*widget.Label) // safe: CreateItem always returns *widget.Label
			lbl.SetText(e.text)
			if e.isError {
				lbl.Importance = widget.DangerImportance
			} else {
				lbl.Importance = widget.MediumImportance
			}
		},
	)
	ui.logList.OnSelected = func(_ widget.ListItemID) {} // prevent selection highlight

	bg := canvas.NewRectangle(colorBGSecondary)
	return container.NewStack(bg,
		container.NewBorder(titleRow, nil, nil, nil, ui.logList),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Mapping CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) addMapping() {
	lp, wp, ok := parsePort(ui.portEntry.Text)
	if !ok {
		ui.showToast(`Formato inválido. Use "3000" ou "3000:3001"`)
		return
	}
	for _, m := range ui.mappings {
		if m.ListenPort == lp {
			ui.showToast(fmt.Sprintf("Porta %d já foi adicionada", lp))
			ui.portEntry.SetText("")
			return
		}
	}
	ui.mappings = append(ui.mappings, PortMapping{ID: newUUID(), ListenPort: lp, WSLPort: wp})
	ui.portEntry.SetText("")
	ui.rebuildMappings()
	saveSettings(ui.mappings)
}

func (ui *appUI) removeMapping(id string) {
	for i, m := range ui.mappings {
		if m.ID == id {
			ui.mappings = append(ui.mappings[:i], ui.mappings[i+1:]...)
			break
		}
	}
	ui.rebuildMappings()
	saveSettings(ui.mappings)
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

// ─────────────────────────────────────────────────────────────────────────────
// Button handlers
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) onStart() {
	if len(ui.mappings) == 0 {
		ui.showToast("Adicione pelo menos uma porta antes de iniciar")
		return
	}
	if ui.engine.State() == StatePaused {
		ui.engine.Resume()
		return
	}
	saveSettings(ui.mappings)
	go ui.engine.Start(ui.mappings)
}

func (ui *appUI) onPause() {
	switch ui.engine.State() {
	case StateRunning:
		ui.engine.Pause()
	case StatePaused:
		ui.engine.Resume()
	}
}

func (ui *appUI) onStop() {
	ui.engine.Stop()
}

// ─────────────────────────────────────────────────────────────────────────────
// State update — always on main goroutine via fyne.Do
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) applyState(state ProxyState, info *WSLInfo) {
	if info != nil {
		ui.lastWSLInfo = info
	}

	switch state {
	case StateRunning:
		ui.statusLabel.Text = "● EXECUTANDO"
		ui.statusLabel.Color = colorGreen
		ui.statusLabel.Refresh()
		if ui.lastWSLInfo != nil {
			mode := "NAT"
			if ui.lastWSLInfo.Mode == "mirrored" {
				mode = "Mirrored"
			}
			ui.wslInfoLabel.SetText(fmt.Sprintf("WSL: %s (%s)", ui.lastWSLInfo.IP, mode))
		}
	case StatePaused:
		ui.statusLabel.Text = "● PAUSADO"
		ui.statusLabel.Color = colorYellow
		ui.statusLabel.Refresh()
	case StateStopped:
		ui.statusLabel.Text = "● PARADO"
		ui.statusLabel.Color = colorRed
		ui.statusLabel.Refresh()
		ui.wslInfoLabel.SetText("WSL não detectado")
		ui.lastWSLInfo = nil
	}

	isStopped := state == StateStopped
	isPaused := state == StatePaused
	isRunning := state == StateRunning

	if isRunning {
		ui.btnStart.Disable()
		ui.btnStart.SetText("▶ Iniciar")
	} else {
		ui.btnStart.Enable()
		if isPaused {
			ui.btnStart.SetText("▶ Retomar")
		} else {
			ui.btnStart.SetText("▶ Iniciar")
		}
	}

	// if isStopped {
	// 	//ui.btnPause.Hide()
	// } else {
	// 	ui.btnPause.Show()
	// 	if isPaused {
	// 		ui.btnPause.SetText("▶ Retomar")
	// 	} else {
	// 		ui.btnPause.SetText("⏸ Pausar")
	// 	}
	// }

	if isStopped {
		ui.btnStop.Disable()
	} else {
		ui.btnStop.Enable()
	}

	locked := isRunning || isPaused
	if locked {
		ui.portEntry.Disable()
		ui.btnAdd.Disable()
	} else {
		ui.portEntry.Enable()
		ui.btnAdd.Enable()
	}

	ui.rebuildMappings()
}

// ─────────────────────────────────────────────────────────────────────────────
// Log — always on main goroutine
// ─────────────────────────────────────────────────────────────────────────────

const maxLogEntries = 500

func (ui *appUI) appendLog(e ConnectionEvent) {
	if len(ui.logEntries) >= maxLogEntries {
		ui.logEntries = ui.logEntries[1:]
	}
	ui.logEntries = append(ui.logEntries, logEntry{
		text: fmt.Sprintf("%s  %-18s  :%d → :%d",
			e.Timestamp.Format("15:04:05"), e.SourceIP, e.ListenPort, e.WSLPort),
	})
	ui.logList.Refresh()
	ui.logList.ScrollToBottom()
	n := len(ui.logEntries)
	if n == 1 {
		ui.logCountLbl.SetText("1 conexão")
	} else {
		ui.logCountLbl.SetText(fmt.Sprintf("%d conexões", n))
	}
}

func (ui *appUI) appendError(listenPort, _ int, msg string) {
	portInfo := ""
	if listenPort > 0 {
		portInfo = fmt.Sprintf(" (porta %d)", listenPort)
	}
	ui.logEntries = append(ui.logEntries, logEntry{
		text:    fmt.Sprintf("%s  ERRO%s: %s", time.Now().Format("15:04:05"), portInfo, msg),
		isError: true,
	})
	ui.logList.Refresh()
	ui.logList.ScrollToBottom()
	ui.showToast(fmt.Sprintf("Erro%s: %s", portInfo, msg))
}

// ─────────────────────────────────────────────────────────────────────────────
// Toast — non-modal popup at bottom-right, auto-dismisses
// ─────────────────────────────────────────────────────────────────────────────

func (ui *appUI) showToast(message string) {
	lbl := widget.NewLabel(message)
	lbl.Wrapping = fyne.TextWrapWord

	bg := canvas.NewRectangle(colorRed)
	content := container.NewStack(bg, container.NewPadded(lbl))

	popup := widget.NewPopUp(content, ui.win.Canvas())
	popW := float32(300)
	popH := float32(64)
	popup.Resize(fyne.NewSize(popW, popH))

	winSize := ui.win.Canvas().Size()
	popup.ShowAtPosition(fyne.NewPos(
		winSize.Width-popW-16,
		winSize.Height-popH-16,
	))

	go func() {
		time.Sleep(3500 * time.Millisecond)
		dispatch(func() { popup.Hide() })
	}()
}
