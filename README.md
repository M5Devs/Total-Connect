# Total Connect

```text
████████╗██████╗ ████████╗█████╗ ██╗      ██████╗ ██████╗ ███╗   ██╗███╗   ██╗███████╗██████╗████████╗
╚══██╔══╝██╔═══██╗╚══██╔══╝██╔══██╗██║     ██╔════╝██╔═══██╗████╗  ██║████╗  ██║██╔════╝██╔════╝╚══██╔══╝
   ██║   ██║   ██║   ██║   ███████║██║     ██║     ██║   ██║██╔██╗ ██║██╔██╗ ██║█████╗  ██║        ██║
   ██║   ██║   ██║   ██║   ██╔══██║██║     ██║     ██║   ██║██║╚██╗██║██║╚██╗██║██╔══╝  ██║        ██║
   ██║   ╚██████╔╝   ██║   ██║  ██║███████╗╚██████╗╚██████╔╝██║ ╚████║██║ ╚████║███████╗╚██████╗   ██║
   ╚═╝    ╚═════╝    ╚═╝   ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═══╝╚══════╝ ╚═════╝   ╚═╝
```

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](go.mod)
[![Platform Support](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)](#)

> **Total Connect** is a modern, high-performance, cross-platform terminal file manager and cloud client built in Go. It blends the iconic dual-pane efficiency of Total Commander and FileZilla with the power and universal backend integration of [`rclone`](https://rclone.org/).

---

## 💡 Vision & Value Proposition

Traditional GUI file managers like FileZilla and Total Commander are powerful but often locked into desktop environments or limited to traditional remote protocols like FTP/SFTP. Command-line utilities like `rclone` offer unparalleled cloud support (S3, Google Drive, OneDrive, Mega, Dropbox, SFTP, SMB, etc.) but lack a responsive, dual-pane interactive interface for fast side-by-side file navigation and management.

**Total Connect** bridges this gap:
- **Dual-Pane Efficiency**: Side-by-side local & remote file navigation inspired by Total Commander.
- **Powered by rclone**: Native access to 40+ cloud storage providers and protocols out-of-the-box.
- **Native Atomic Operations**: True atomic file/directory moves and server-side transfers without naive copy-then-delete risks.
- **Cross-Platform & Lightweight**: Compiles to a single zero-dependency binary for Linux, macOS, and Windows.
- **Terminal-First UX**: Built using Charmbracelet (`bubbletea`, `lipgloss`, `bubbles`) for smooth, reactive terminal interaction with keyboard-driven hotkeys.

---

## 🏗️ Architecture Overview

Total Connect is architected with clear boundaries separating the presentation interfaces, engine orchestration, configuration management, and core storage driver layer:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                          Interfaces / Consumers                        │
├───────────────────────────────┬─────────────────────────┬──────────────┤
│    cmd/tc-tui (Bubbletea TUI) │  cmd/tc-cli (CLI Tool)   │ Web Daemon   │
│                               │                         │ (tc-server)  │
└───────────────┬───────────────┴────────────┬────────────┴──────────────┘
                │                            │
                ▼                            ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   Session Management (`internal/core`)                  │
│                     SessionManager & State Handlers                     │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│                 Storage Engine Layer (`internal/core`)                 │
│               StorageEngine Interface (Copy, Move, Delete)              │
├────────────────────────────────────────────────────────────────────────┤
│                    RcloneEngine (`rclone_ops.go`)                      │
│             rclone/fs, operations, sync, and accounting               │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│              Config & Storage Backends (`internal/config`)             │
│            Cross-Platform rclone.conf resolution & parsing             │
│     (Local FS, S3, Google Drive, SFTP, SMB, Mega, Dropbox, etc.)      │
└────────────────────────────────────────────────────────────────────────┘
```

### Module Breakdown:
- **`cmd/tc-tui`**: The interactive Terminal User Interface entry point built on `bubbletea`.
- **`cmd/tc-cli`**: Command-line execution tool for scriptable tasks and headless operations.
- **`cmd/tc-server`**: Web daemon serving the web dashboard and REST API for remote management.
- **`internal/tui`**: UI components including dual navigation panes, reactive status bar with active transfer progress, remote selection modal, and shortcut cheat sheet.
- **`internal/core`**: `StorageEngine` abstraction layer and `RcloneEngine` implementation utilizing rclone operations natively for atomic file/directory manipulation and progress callbacks.
- **`internal/config`**: Unified cross-platform config discovery resolving `rclone.conf` paths across Linux, macOS, and Windows.
- **`internal/models`**: Standardized data models (`FileItem`, `RemoteItem`, `Progress`).

---

## ⌨️ Keyboard Shortcuts

Total Connect is designed for speed and productivity with simple keyboard shortcuts:

| Shortcut | Action | Description |
| :--- | :--- | :--- |
| `Tab` | **Switch Pane** | Toggle focus between Left and Right panes |
| `Up` / `k` | **Cursor Up** | Move selection up by one row |
| `Down` / `j` | **Cursor Down** | Move selection down by one row |
| `PgUp` / `PgDown` | **Page Up / Down** | Scroll active pane by page height |
| `Enter` | **Open / Enter** | Enter selected directory or open parent `..` |
| `Backspace` | **Parent Directory** | Navigate up to parent directory |
| `F5` / `c` | **Copy** | Copy selected item to opposite pane with progress bar |
| `F6` / `m` | **Move** | Move selected item natively to opposite pane |
| `F7` / `n` | **Mkdir** | Open prompt to create new directory |
| `F8` / `d` | **Delete** | Delete selected file or directory with confirmation |
| `r` | **Select Remote** | Open storage backend selector modal |
| `Ctrl+R` | **Refresh** | Reload active pane directory contents |
| `?` | **Help** | Toggle Keyboard Shortcut Cheat Sheet modal |
| `q` / `Ctrl+C` | **Quit** | Exit Total Connect |

---

## 🚀 Quick Start

### Prerequisites
- **Go**: Version 1.22 or higher.
- **Git**: For cloning repository.

### Installation & Run

1. **Clone the repository:**
   ```bash
   git clone https://github.com/M5Devs/Total-Connect.git
   cd Total-Connect
   ```

2. **Run the TUI directly:**
   ```bash
   go run ./cmd/tc-tui
   ```

3. **Build binaries:**
   ```bash
   # Build TUI application
   go build -o bin/tc-tui ./cmd/tc-tui

   # Build CLI tool
   go build -o bin/tc-cli ./cmd/tc-cli

   # Build Web Daemon
   go build -o bin/tc-server ./cmd/tc-server
   ```

4. **Run Unit Tests:**
   ```bash
   go test -v ./...
   ```

---

## 🔒 Security Considerations

### 🔑 Rclone Obfuscation vs. Encryption
- **Obfuscation Warning**: `rclone obscure` uses weak password obfuscation to prevent casual glance reading of passwords in `rclone.conf`. It is **not encryption** and does not protect credentials against anyone with local filesystem access.
- **At-Rest Password Encryption**: If you require secure credential storage at rest, encrypt your `rclone.conf` using a master password (`rclone config password`). At runtime, supply your master key via the standard `RCLONE_CONFIG_PASS` environment variable:
  ```bash
  export RCLONE_CONFIG_PASS="your-master-password"
  ./bin/tc-server
  ```

### 🌐 Web Daemon (`tc-server`) Exposure
- **Secure Defaults**: By default, `tc-server` listens exclusively on loopback (`127.0.0.1:8080`) to prevent accidental public exposure.
- **Authentication**: When exposing `tc-server` across networks or on public interfaces (e.g. `0.0.0.0`), authentication is strictly enforced. Set HTTP Basic Auth credentials using the `-auth` flag or `TC_AUTH` environment variable:
  ```bash
  # Using command line flag
  ./bin/tc-server -host 0.0.0.0 -auth "admin:securepassword"

  # Using environment variable
  export TC_AUTH="admin:securepassword"
  ./bin/tc-server -host 0.0.0.0
  ```
- **Bypassing Auth (Insecure)**: Binding to a public/non-loopback interface without authentication will cause `tc-server` to immediately terminate with a security error. To bypass this check for isolated test setups, explicitly pass `-insecure-no-auth`:
  ```bash
  ./bin/tc-server -host 0.0.0.0 -insecure-no-auth
  ```
- **Reverse Proxy & TLS**: In production environments, keep `tc-server` bound to `127.0.0.1` and expose it securely behind a reverse proxy (such as Nginx, Caddy, or Cloudflare Tunnel) with HTTPS/TLS enabled.

#### Example Caddy Reverse Proxy Configuration:
```caddy
files.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

#### Example Nginx Reverse Proxy Configuration:
```nginx
server {
    listen 443 ssl http2;
    server_name files.example.com;

    ssl_certificate /etc/letsencrypt/live/files.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/files.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 📜 License

Total Connect is open-source software licensed under the **GNU General Public License v3.0 (GPLv3)**. See the [LICENSE](LICENSE) file for details.
