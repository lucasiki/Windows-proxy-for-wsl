#!/usr/bin/env python3
"""WSL Proxy — GUI em Python/Tkinter"""

import asyncio
import json
import os
import queue
import re
import socket
import subprocess
import sys
import threading
import tkinter as tk
import uuid
from datetime import datetime

# ── Cores (mesmas variáveis do CSS) ──────────────────────────────────────────
BG_PRIMARY   = '#1a1a2e'
BG_SECONDARY = '#16213e'
BG_SURFACE   = '#0d2137'
BG_INPUT     = '#0a1628'
BORDER       = '#1e3a5f'
ACCENT       = '#4f8ef7'
GREEN        = '#4caf50'
YELLOW       = '#ffb300'
RED          = '#e94560'
TEXT_PRIMARY = '#e8eaf6'
TEXT_MUTED   = '#7986a8'
TEXT_DIM     = '#4a5568'


# ── Paths ────────────────────────────────────────────────────────────────────

def _base_dir():
    """Diretório base: ao lado do .exe quando frozen, senão ao lado do .py."""
    if getattr(sys, 'frozen', False):
        return os.path.dirname(sys.executable)
    return os.path.dirname(os.path.abspath(__file__))


def resource_path(rel):
    """Caminho para arquivo bundled pelo PyInstaller (--add-data)."""
    if getattr(sys, 'frozen', False):
        base = sys._MEIPASS  # type: ignore[attr-defined]
    else:
        base = os.path.dirname(os.path.abspath(__file__))
    return os.path.join(base, rel)


SETTINGS_FILE = os.path.join(_base_dir(), 'wsl_proxy_settings.json')


# ── Configurações ─────────────────────────────────────────────────────────────

def load_settings() -> dict:
    try:
        with open(SETTINGS_FILE, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception:
        return {'port_mappings': []}


def save_settings(data: dict):
    try:
        with open(SETTINGS_FILE, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
    except Exception:
        pass


# ── Detecção do WSL ───────────────────────────────────────────────────────────

def get_target_info():
    """Retorna (wsl_ip_display, mode, target_ip) ou (None, None, None) se falhar."""
    try:
        result = subprocess.run(
            ['wsl', 'hostname', '-I'],
            capture_output=True, text=True, timeout=5
        )
        parts = result.stdout.strip().split()
        if not parts:
            return None, None, None
        wsl_ip = parts[0]
        try:
            local_ips = socket.gethostbyname_ex(socket.gethostname())[2]
        except Exception:
            local_ips = []
        if wsl_ip in local_ips:
            return wsl_ip, 'mirrored', '127.0.0.1'
        return wsl_ip, 'NAT', wsl_ip
    except Exception:
        return None, None, None


# ── Proxy assíncrono ──────────────────────────────────────────────────────────

async def _pipe(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    try:
        while not reader.at_eof():
            data = await reader.read(4096)
            if not data:
                break
            writer.write(data)
            await writer.drain()
    except Exception:
        pass
    finally:
        try:
            writer.close()
        except Exception:
            pass


class _ProxyServer:
    """Gerencia um par listen_port → wsl_port."""

    def __init__(self, listen_port: int, wsl_port: int, target_ip: str,
                 on_connection, on_error):
        self.listen_port = listen_port
        self.wsl_port = wsl_port
        self.target_ip = target_ip
        self.on_connection = on_connection
        self.on_error = on_error
        self.paused = False
        self._servers: list[asyncio.AbstractServer] = []

    async def _handle(self, lr: asyncio.StreamReader, lw: asyncio.StreamWriter):
        if self.paused:
            lw.close()
            return
        src_ip = lw.get_extra_info('peername', ('unknown', 0))[0]
        try:
            rr, rw = await asyncio.open_connection(self.target_ip, self.wsl_port)
            self.on_connection({
                'timestamp': datetime.now().isoformat(),
                'source_ip': src_ip,
                'listen_port': self.listen_port,
                'wsl_port': self.wsl_port,
            })
            await asyncio.gather(_pipe(lr, rw), _pipe(rr, lw))
        except Exception as e:
            self.on_error(self.listen_port, self.wsl_port, str(e))
        finally:
            try:
                lw.close()
            except Exception:
                pass

    async def start(self):
        try:
            s4 = await asyncio.start_server(self._handle, '0.0.0.0', self.listen_port)
            self._servers.append(s4)
        except Exception as e:
            self.on_error(self.listen_port, self.wsl_port, str(e))
            raise
        try:
            s6 = await asyncio.start_server(self._handle, '::', self.listen_port)
            self._servers.append(s6)
        except Exception:
            pass  # IPv6 opcional

    async def stop(self):
        for s in self._servers:
            s.close()
            try:
                await s.wait_closed()
            except Exception:
                pass
        self._servers.clear()

    def pause(self):
        self.paused = True

    def resume(self):
        self.paused = False


class ProxyEngine:
    """Orquestra os _ProxyServer numa thread de background."""

    def __init__(self, on_connection, on_error, on_state_change):
        self.on_connection = on_connection
        self.on_error = on_error
        self.on_state_change = on_state_change
        self._loop: asyncio.AbstractEventLoop | None = None
        self._thread: threading.Thread | None = None
        self._ready = threading.Event()
        self._servers: list[_ProxyServer] = []
        self.state = 'stopped'
        self.wsl_ip: str | None = None
        self.wsl_mode: str | None = None
        self._ensure_loop()

    def _run_loop(self):
        self._loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self._loop)
        self._ready.set()
        self._loop.run_forever()

    def _ensure_loop(self):
        if self._thread is None or not self._thread.is_alive():
            self._ready.clear()
            self._loop = None
            self._thread = threading.Thread(target=self._run_loop, daemon=True, name='ProxyLoop')
            self._thread.start()
            self._ready.wait()

    def start(self, port_mappings: list[dict]):
        wsl_ip, wsl_mode, target_ip = get_target_info()
        if target_ip is None:
            self.on_error(0, 0, 'WSL não detectado. Certifique-se de que o WSL está rodando.')
            return

        self.wsl_ip = wsl_ip
        self.wsl_mode = wsl_mode

        async def _start():
            for m in port_mappings:
                srv = _ProxyServer(
                    m['listen_port'], m['wsl_port'], target_ip,
                    self.on_connection,
                    lambda lp, wp, msg: self.on_error(lp, wp, msg),
                )
                try:
                    await srv.start()
                    self._servers.append(srv)
                except Exception:
                    pass  # erro já emitido dentro do _ProxyServer.start()
            self.state = 'running'
            self.on_state_change(self.state, self.wsl_ip, self.wsl_mode)

        self._ensure_loop()
        asyncio.run_coroutine_threadsafe(_start(), self._loop)  # type: ignore[arg-type]

    def pause(self):
        if self.state != 'running':
            return
        for s in self._servers:
            s.pause()
        self.state = 'paused'
        self.on_state_change(self.state, self.wsl_ip, self.wsl_mode)

    def resume(self):
        if self.state != 'paused':
            return
        for s in self._servers:
            s.resume()
        self.state = 'running'
        self.on_state_change(self.state, self.wsl_ip, self.wsl_mode)

    def stop(self):
        if self.state == 'stopped':
            return
        old_servers = self._servers[:]
        self._servers.clear()
        self.state = 'stopped'
        self.wsl_ip = None
        self.wsl_mode = None
        self.on_state_change('stopped', None, None)

        async def _cleanup():
            for s in old_servers:
                await s.stop()

        if self._loop:
            asyncio.run_coroutine_threadsafe(_cleanup(), self._loop)  # type: ignore[arg-type]


# ── GUI ───────────────────────────────────────────────────────────────────────

class App(tk.Tk):
    def __init__(self):
        super().__init__()
        self.title('WSL Proxy')
        self.geometry('620x540')
        self.minsize(520, 460)
        self.configure(bg=BG_PRIMARY)
        self._set_icon()

        self.port_mappings: list[dict] = []
        self.proxy_state = 'stopped'
        self.log_count = 0
        self._q: queue.Queue = queue.Queue()

        self._engine = ProxyEngine(
            on_connection=lambda e: self._q.put(('log', e)),
            on_error=lambda lp, wp, msg: self._q.put(('error', lp, wp, msg)),
            on_state_change=lambda s, ip, m: self._q.put(('state', s, ip, m)),
        )

        self._build_ui()
        self._load_settings()
        self._poll()

    # ── Ícone ────────────────────────────────────────────────────────────────

    def _set_icon(self):
        for rel in ('electron-app/assets/icon.ico', 'assets/icon.ico'):
            path = resource_path(rel)
            if os.path.exists(path):
                try:
                    self.iconbitmap(path)
                except Exception:
                    pass
                return

    # ── Construção da UI ─────────────────────────────────────────────────────

    def _build_ui(self):
        self._build_header()
        content = tk.Frame(self, bg=BG_PRIMARY)
        content.pack(fill='both', expand=True, padx=14, pady=10)
        self._build_port_panel(content)
        self._build_log_panel(content)

    def _build_header(self):
        hdr = tk.Frame(self, bg=BG_SECONDARY, height=50)
        hdr.pack(fill='x')
        hdr.pack_propagate(False)

        # Logo
        c = tk.Canvas(hdr, width=28, height=28, bg=BG_SECONDARY, highlightthickness=0)
        c.pack(side='left', padx=(14, 8), pady=11)
        c.create_oval(1, 1, 27, 27, fill='#0d2137', outline='#1e3a5f', width=1)
        c.create_line(5, 9,  16, 9,  fill=ACCENT,  width=2, capstyle='round')
        c.create_line(5, 14, 21, 14, fill=ACCENT,  width=2, capstyle='round')
        c.create_line(5, 19, 18, 19, fill=ACCENT,  width=2, capstyle='round')
        c.create_oval(19, 7, 25, 13, fill=GREEN, outline='')

        tk.Label(hdr, text='WSL Proxy', bg=BG_SECONDARY, fg=TEXT_PRIMARY,
                 font=('Segoe UI', 13, 'bold')).pack(side='left')

        right = tk.Frame(hdr, bg=BG_SECONDARY)
        right.pack(side='right', padx=14)

        self._wsl_info_var = tk.StringVar(value='WSL não detectado')
        tk.Label(right, textvariable=self._wsl_info_var, bg=BG_SECONDARY,
                 fg=TEXT_MUTED, font=('Segoe UI', 10)).pack(side='left', padx=(0, 10))

        self._status_lbl = tk.Label(right, text='● PARADO', bg=BG_SECONDARY,
                                     fg=TEXT_MUTED, font=('Segoe UI', 9, 'bold'))
        self._status_lbl.pack(side='left')

    def _panel(self, parent):
        f = tk.Frame(parent, bg=BG_SECONDARY, highlightbackground=BORDER, highlightthickness=1)
        return f

    def _panel_header(self, panel, title):
        hdr = tk.Frame(panel, bg=BG_SURFACE, height=32)
        hdr.pack(fill='x')
        hdr.pack_propagate(False)
        tk.Label(hdr, text=title, bg=BG_SURFACE, fg=TEXT_MUTED,
                 font=('Segoe UI', 9, 'bold')).pack(side='left', padx=12, pady=7)
        return hdr

    def _build_port_panel(self, parent):
        panel = self._panel(parent)
        panel.pack(fill='x', pady=(0, 10))

        self._panel_header(panel, 'MAPEAMENTO DE PORTAS')

        # Input row
        row = tk.Frame(panel, bg=BG_SECONDARY)
        row.pack(fill='x', padx=10, pady=8)

        self._port_entry = tk.Entry(
            row, bg=BG_INPUT, fg=TEXT_DIM, insertbackground=TEXT_PRIMARY,
            relief='flat', highlightbackground=BORDER, highlightthickness=1,
            font=('Consolas', 11),
        )
        self._port_entry.pack(side='left', fill='x', expand=True, ipady=5, padx=(0, 8))
        self._port_entry.insert(0, 'ex: 3000  ou  3000:3001  (Windows:WSL)')
        self._placeholder_active = True
        self._port_entry.bind('<FocusIn>',  self._on_entry_focus_in)
        self._port_entry.bind('<FocusOut>', self._on_entry_focus_out)
        self._port_entry.bind('<Return>',   lambda _e: self._add_mapping())

        self._btn_add = self._btn(row, '+ Adicionar', ACCENT, 'white', self._add_mapping)
        self._btn_add.pack(side='left')

        # Mapping list
        self._map_frame = tk.Frame(panel, bg=BG_SECONDARY)
        self._map_frame.pack(fill='x')

        # Controls
        ctrl = tk.Frame(panel, bg=BG_SURFACE, height=42)
        ctrl.pack(fill='x')
        ctrl.pack_propagate(False)

        bf = tk.Frame(ctrl, bg=BG_SURFACE)
        bf.pack(side='left', padx=10, pady=8)

        self._btn_start = self._btn(bf, '▶ Iniciar', ACCENT, 'white', self._on_start)
        self._btn_start.grid(row=0, column=0, padx=(0, 6))

        self._btn_pause = self._btn(bf, '⏸ Pausar', BG_SECONDARY, YELLOW, self._on_pause,
                                     border_color='#ffb30066')
        self._btn_pause.grid(row=0, column=1, padx=(0, 6))
        self._btn_pause.grid_remove()

        self._btn_stop = self._btn(bf, '■ Parar', BG_SECONDARY, RED, self._on_stop,
                                    border_color='#e9456066', state='disabled')
        self._btn_stop.grid(row=0, column=2)

    def _build_log_panel(self, parent):
        panel = self._panel(parent)
        panel.pack(fill='both', expand=True)

        hdr = self._panel_header(panel, 'LOG DE CONEXÕES')

        right = tk.Frame(hdr, bg=BG_SURFACE)
        right.pack(side='right', padx=8)

        self._log_count_var = tk.StringVar(value='')
        tk.Label(right, textvariable=self._log_count_var, bg=BG_SURFACE,
                 fg=TEXT_DIM, font=('Segoe UI', 9)).pack(side='left', padx=(0, 6))
        self._btn(right, 'Limpar', BG_SURFACE, TEXT_MUTED, self._clear_log,
                  border_color=BORDER, font_size=9, padx=6, pady=2).pack(side='left', pady=5)

        container = tk.Frame(panel, bg=BG_SECONDARY)
        container.pack(fill='both', expand=True)

        self._log_text = tk.Text(
            container, bg=BG_SECONDARY, fg=TEXT_PRIMARY,
            relief='flat', font=('Consolas', 10),
            state='disabled', wrap='none', cursor='arrow',
            highlightthickness=0,
        )
        vsb = tk.Scrollbar(container, command=self._log_text.yview,
                           bg=BG_SURFACE, troughcolor=BG_SECONDARY,
                           activebackground=BORDER, relief='flat', width=10)
        self._log_text.configure(yscrollcommand=vsb.set)
        vsb.pack(side='right', fill='y')
        self._log_text.pack(fill='both', expand=True)

        self._log_text.tag_configure('time',  foreground=TEXT_DIM)
        self._log_text.tag_configure('ip',    foreground=TEXT_MUTED)
        self._log_text.tag_configure('ports', foreground=ACCENT)
        self._log_text.tag_configure('error', foreground=RED)

    # ── Botão helper ─────────────────────────────────────────────────────────

    def _btn(self, parent, text, bg, fg, cmd, border_color=None,
             state='normal', font_size=9, padx=12, pady=4):
        kw = dict(
            text=text, bg=bg, fg=fg, activebackground=bg, activeforeground=fg,
            relief='flat', font=('Segoe UI', font_size, 'bold'),
            cursor='hand2', command=cmd, padx=padx, pady=pady, state=state,
        )
        if border_color:
            kw['highlightbackground'] = border_color
            kw['highlightthickness'] = 1
        return tk.Button(parent, **kw)

    # ── Placeholder ──────────────────────────────────────────────────────────

    def _on_entry_focus_in(self, _e=None):
        if self._placeholder_active:
            self._port_entry.delete(0, 'end')
            self._port_entry.config(fg=TEXT_PRIMARY)
            self._placeholder_active = False

    def _on_entry_focus_out(self, _e=None):
        if not self._port_entry.get():
            self._port_entry.insert(0, 'ex: 3000  ou  3000:3001  (Windows:WSL)')
            self._port_entry.config(fg=TEXT_DIM)
            self._placeholder_active = True

    # ── Port mappings ─────────────────────────────────────────────────────────

    def _add_mapping(self):
        if self._placeholder_active:
            self._show_toast('Informe uma porta antes de adicionar')
            return
        raw = self._port_entry.get().strip()
        parsed = self._parse_port(raw)
        if not parsed:
            self._show_toast('Formato inválido. Use "3000" ou "3000:3001"')
            return
        lp, wp = parsed
        if any(m['listen_port'] == lp for m in self.port_mappings):
            self._show_toast(f'Porta {lp} já foi adicionada')
            self._port_entry.delete(0, 'end')
            self._on_entry_focus_out()
            return
        self.port_mappings.append({'id': str(uuid.uuid4()), 'listen_port': lp, 'wsl_port': wp})
        self._port_entry.delete(0, 'end')
        self._on_entry_focus_out()
        self._render_mappings()
        self._save_settings()

    @staticmethod
    def _parse_port(raw: str):
        m = re.match(r'^(\d+)(?::(\d+))?$', raw)
        if not m:
            return None
        lp = int(m.group(1))
        wp = int(m.group(2) or m.group(1))
        if not (1 <= lp <= 65535 and 1 <= wp <= 65535):
            return None
        return lp, wp

    def _remove_mapping(self, mid: str):
        self.port_mappings = [m for m in self.port_mappings if m['id'] != mid]
        self._render_mappings()
        self._save_settings()

    def _render_mappings(self):
        for w in self._map_frame.winfo_children():
            w.destroy()
        if not self.port_mappings:
            tk.Label(self._map_frame,
                     text='Nenhuma porta configurada — adicione uma acima',
                     bg=BG_SECONDARY, fg=TEXT_DIM, font=('Segoe UI', 10)
                     ).pack(pady=10)
            return

        locked = self.proxy_state in ('running', 'paused')
        for m in self.port_mappings:
            row = tk.Frame(self._map_frame, bg=BG_SECONDARY)
            row.pack(fill='x', padx=10, pady=2)
            tk.Label(row, text=f":{m['listen_port']} → :{m['wsl_port']}",
                     bg=BG_SECONDARY, fg=ACCENT, font=('Consolas', 11)
                     ).pack(side='left')
            tk.Label(row, text='Windows → WSL',
                     bg=BG_SECONDARY, fg=TEXT_MUTED, font=('Segoe UI', 9)
                     ).pack(side='left', padx=10)
            mid = m['id']
            rb = tk.Button(row, text='✕', bg=BG_SECONDARY, fg=TEXT_MUTED,
                           activebackground=BG_SECONDARY, activeforeground=RED,
                           relief='flat', cursor='hand2', font=('Segoe UI', 11),
                           command=lambda i=mid: self._remove_mapping(i),
                           state='disabled' if locked else 'normal')
            rb.pack(side='right')

    # ── Controls ─────────────────────────────────────────────────────────────

    def _on_start(self):
        if not self.port_mappings:
            self._show_toast('Adicione pelo menos uma porta antes de iniciar')
            return
        if self.proxy_state == 'paused':
            self._engine.resume()
            return
        self._save_settings()
        self._engine.start(self.port_mappings)

    def _on_pause(self):
        if self.proxy_state == 'running':
            self._engine.pause()
        elif self.proxy_state == 'paused':
            self._engine.resume()

    def _on_stop(self):
        self._engine.stop()

    # ── Atualização de estado ─────────────────────────────────────────────────

    def _apply_state(self, state: str, wsl_ip, wsl_mode):
        self.proxy_state = state

        if state == 'running':
            self._status_lbl.config(text='● EXECUTANDO', fg=GREEN)
        elif state == 'paused':
            self._status_lbl.config(text='● PAUSADO', fg=YELLOW)
        else:
            self._status_lbl.config(text='● PARADO', fg=TEXT_MUTED)

        if wsl_ip:
            mode_lbl = 'Mirrored' if wsl_mode == 'mirrored' else 'NAT'
            self._wsl_info_var.set(f'WSL: {wsl_ip} ({mode_lbl})')
        else:
            self._wsl_info_var.set('WSL não detectado')

        is_stopped = state == 'stopped'
        is_paused  = state == 'paused'
        is_running = state == 'running'

        self._btn_start.config(
            state='disabled' if is_running else 'normal',
            text='▶ Retomar' if is_paused else '▶ Iniciar',
        )
        self._btn_stop.config(state='disabled' if is_stopped else 'normal')

        if is_stopped:
            self._btn_pause.grid_remove()
        else:
            self._btn_pause.grid()
            self._btn_pause.config(text='▶ Retomar' if is_paused else '⏸ Pausar')

        locked = is_running or is_paused
        self._port_entry.config(state='disabled' if locked else 'normal')
        self._btn_add.config(state='disabled' if locked else 'normal')
        self._render_mappings()

    # ── Log ───────────────────────────────────────────────────────────────────

    def _log_write(self, func):
        self._log_text.config(state='normal')
        func()
        self._log_text.config(state='disabled')
        self._log_text.see('end')

    def _append_log(self, entry: dict):
        MAX = 500
        if self.log_count >= MAX:
            self._log_write(lambda: self._log_text.delete('1.0', '2.0'))
            self.log_count = MAX - 1

        dt = datetime.fromisoformat(entry['timestamp']).strftime('%H:%M:%S')

        def _write():
            self._log_text.insert('end', f'{dt:<10}',  'time')
            self._log_text.insert('end', f"{entry['source_ip']:<22}", 'ip')
            self._log_text.insert('end',
                f":{entry['listen_port']} → :{entry['wsl_port']}\n", 'ports')

        self._log_write(_write)
        self.log_count += 1
        n = self.log_count
        self._log_count_var.set(f"{n} conex{'ões' if n != 1 else 'ão'}")

    def _append_error(self, listen_port: int, wsl_port: int, msg: str):
        dt = datetime.now().strftime('%H:%M:%S')
        port_info = f':{listen_port}' if listen_port else ''

        def _write():
            self._log_text.insert('end', f'{dt:<10}', 'time')
            self._log_text.insert('end', f'ERRO{port_info}: {msg}\n', 'error')

        self._log_write(_write)

    def _clear_log(self):
        self._log_write(lambda: self._log_text.delete('1.0', 'end'))
        self.log_count = 0
        self._log_count_var.set('')

    # ── Poll da fila de atualizações ─────────────────────────────────────────

    def _poll(self):
        try:
            while True:
                item = self._q.get_nowait()
                kind = item[0]
                if kind == 'log':
                    self._append_log(item[1])
                elif kind == 'error':
                    _, lp, wp, msg = item
                    self._append_error(lp, wp, msg)
                    self._show_toast(f'Porta {lp}: {msg}' if lp else msg)
                elif kind == 'state':
                    _, state, ip, mode = item
                    self._apply_state(state, ip, mode)
        except queue.Empty:
            pass
        self.after(50, self._poll)

    # ── Toast ─────────────────────────────────────────────────────────────────

    def _show_toast(self, message: str, duration_ms: int = 3500):
        toast = tk.Toplevel(self)
        toast.overrideredirect(True)
        toast.attributes('-topmost', True)
        tk.Label(toast, text=message, bg='#c0263f', fg='white',
                 font=('Segoe UI', 10), padx=14, pady=8,
                 wraplength=300).pack()
        self.update_idletasks()
        x = self.winfo_x() + self.winfo_width()  - 330
        y = self.winfo_y() + self.winfo_height() - 80
        toast.geometry(f'+{max(x, 0)}+{max(y, 0)}')
        toast.after(duration_ms, toast.destroy)

    # ── Settings ──────────────────────────────────────────────────────────────

    def _load_settings(self):
        settings = load_settings()
        self.port_mappings = settings.get('port_mappings', [])
        for m in self.port_mappings:
            if 'id' not in m:
                m['id'] = str(uuid.uuid4())
        self._render_mappings()

    def _save_settings(self):
        save_settings({'port_mappings': self.port_mappings})


# ── Entry point ───────────────────────────────────────────────────────────────

if __name__ == '__main__':
    app = App()
    app.mainloop()
