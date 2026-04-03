<p align="center">
  <br>
  <img src=".github/assets/banner.png" alt="Nextcloud Installer" width="700">
  <br>
  <br>
</p>

<h1 align="center">Raspberry Pi Nextcloud Installer</h1>

<p align="center">
  <strong>A beautiful terminal UI that turns your Raspberry Pi into a personal cloud server.</strong>
</p>

<p align="center">
  <a href="#features">Features</a> &bull;
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#screenshots">Screenshots</a> &bull;
  <a href="#usage">Usage</a> &bull;
  <a href="#building">Building</a> &bull;
  <a href="#architecture">Architecture</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Bubbletea-TUI-FF5F87?style=flat-square" alt="Bubbletea">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Platform-Raspberry%20Pi-C51A4A?style=flat-square&logo=raspberrypi" alt="Raspberry Pi">
</p>

---

Stop wrestling with copy-pasted terminal commands. This installer walks you through the entire Nextcloud setup — from a fresh Raspberry Pi to a fully working private cloud — in a single guided session. Every step is interactive, validated, and reversible.

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss) for a native terminal experience that feels as polished as a desktop app.

## Screenshots

<p align="center">
  <img src=".github/assets/screenshot-welcome.png" alt="Welcome screen" width="700">
  <br>
  <em>Welcome screen with system detection and prerequisite checks</em>
</p>

<p align="center">
  <img src=".github/assets/screenshot-database.png" alt="Database setup" width="700">
  <br>
  <em>Interactive database configuration with input validation</em>
</p>

<p align="center">
  <img src=".github/assets/screenshot-progress.png" alt="Installation progress" width="700">
  <br>
  <em>Real-time progress with animated spinners</em>
</p>

## Features

### Guided Setup
11 steps covering the complete Nextcloud stack — nothing left to figure out on your own:

| Step | What it does | Required |
|------|-------------|----------|
| **Welcome** | System detection, prerequisite checks, keybinding reference | Yes |
| **Apache & PHP** | Installs Apache2, adds Sury PHP 8.4 repo, installs PHP + 12 extensions, enables modules | Yes |
| **Database** | Installs MariaDB, creates database, user, and grants privileges | Yes |
| **Download** | Downloads latest Nextcloud, extracts, sets permissions | Yes |
| **Apache Config** | Writes virtual host config (directory or domain mode), enables site | Yes |
| **Web Setup** | Opens Nextcloud in your browser for first-run wizard | Yes |
| **Move Data** | Relocates data directory for security or external storage | Optional |
| **Upload Limit** | Increases PHP upload cap from 2 MB to your chosen size | Optional |
| **SSL / HTTPS** | Generates self-signed certificate, enables SSL, optional HTTP redirect | Optional |
| **Port Forwarding** | Adds trusted domains, shows router config instructions | Optional |
| **Complete** | Summary of everything configured, next steps | Yes |

### Vim-Style Navigation
Move through the interface the way you move through code:

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll content, navigate lists |
| `h` / `l` | Switch focus between buttons |
| `Enter` | Confirm selection |
| `Esc` | Go back to previous field, or skip optional steps |
| `y` / `n` | Quick-select Yes/No |
| `Ctrl+D` / `Ctrl+U` | Half-page scroll |
| `Ctrl+C` | Quit |

### Smart Defaults
Every input comes pre-filled with a sensible value. Just hit Enter to accept, or type to customize:

- Database name: `nextclouddb`
- Database user: `nextclouduser`
- Upload limit: `1024M`
- Data directory: `/var/nextcloud/data`
- Apache mode: Directory (`/nextcloud`)

### Terminal Image Rendering
Tutorial screenshots are rendered directly in your terminal using the best available method:

| Mode | How it works | Quality |
|------|-------------|---------|
| **Sixel** | Native pixel graphics in supported terminals (foot, wezterm, mlterm) | Best |
| **Unicode** | Colored block characters via [chafa](https://hpjansson.org/chafa/) | Good |
| **None** | Text placeholders with image dimensions | Fallback |

Auto-detected at startup. Override with `--images=sixel|unicode|none`.

### Bulletproof Execution
- **Input validation** on all fields (MySQL identifiers, PHP sizes, domain names, paths)
- **Sudo password** collected once, cached for the session, never written to disk
- **Error recovery** at every step — retry, skip (if optional), or quit
- **Async image loading** with 15-second timeout — never blocks the UI
- **Prerequisite warnings** for missing apt, systemd, low disk space, non-Debian systems

## Quick Start

### Pre-built Binary

```bash
# Download the latest release
curl -fsSL https://github.com/tsch0hnny/rpi-nextcloud/releases/latest/download/nextcloud-installer-linux-arm64 -o nextcloud-installer
chmod +x nextcloud-installer

# Run it
sudo ./nextcloud-installer
```

### From Source

```bash
git clone https://github.com/tsch0hnny/rpi-nextcloud.git
cd rpi-nextcloud
go build -o nextcloud-installer .
sudo ./nextcloud-installer
```

> **Note:** Requires Go 1.22 or later. The binary must be run with `sudo` since it installs system packages and modifies Apache/PHP configuration.

## Usage

```
Usage: nextcloud-installer [flags]

Flags:
  --images string    Image rendering mode (default "auto")
                     Options: sixel, unicode, none, auto
```

### Examples

```bash
# Standard run with auto-detected image rendering
sudo ./nextcloud-installer

# Force unicode art images (useful over SSH)
sudo ./nextcloud-installer --images=unicode

# Disable images entirely for minimal terminals
sudo ./nextcloud-installer --images=none
```

### What You Need

- **Raspberry Pi** (any model) running Raspberry Pi OS, Debian, or Ubuntu
- **Internet connection** for downloading packages and Nextcloud
- **~2 GB free disk space** (more if you plan to store files)
- **SSH or terminal access** to your Pi

### What Gets Installed

| Package | Purpose |
|---------|---------|
| Apache2 | Web server |
| PHP 8.4 + extensions | Runtime for Nextcloud |
| MariaDB | Database server |
| Nextcloud (latest) | The cloud platform itself |
| OpenSSL | Certificate generation (if SSL enabled) |

## Building

### Requirements
- Go 1.22+

### Build

```bash
go build -o nextcloud-installer .
```

### Cross-compile for Raspberry Pi

```bash
# ARM64 (Pi 4, Pi 5)
GOOS=linux GOARCH=arm64 go build -o nextcloud-installer-arm64 .

# ARM32 (Pi 3, Pi Zero 2)
GOOS=linux GOARCH=arm GOARM=7 go build -o nextcloud-installer-arm32 .
```

### Verify

```bash
go vet ./...
```

## Architecture

```
main.go                          Entry point, flag parsing
internal/
  style/
    theme.go                     Nextcloud color palette, lipgloss styles
    keys.go                      Vim-style keybinding definitions
  ui/
    components.go                Reusable: code blocks, status lines, boxes, logo
    confirm.go                   Yes/No with h/l focus, y/n quick keys
    input.go                     Text input with validation support
    password.go                  Masked password field
    selector.go                  j/k choice list
    spinner.go                   Animated progress indicator
    image.go                     Async sixel/chafa/none image rendering
    validate.go                  Input validators (MySQL, PHP size, domain, path)
  exec/
    runner.go                    Sudo-aware command execution
    detect.go                    System info (IP, disk, OS, services)
  app/
    app.go                       Top-level Bubbletea model, step orchestrator
  steps/
    steps.go                     Step interface, shared state
    welcome.go                   Welcome + system checks
    apache_php.go                Apache + PHP repo + modules
    mysql.go                     MariaDB + database setup
    download.go                  Nextcloud download + extract
    apache_conf.go               Apache virtual host config
    web_setup.go                 Browser-based first-run wizard
    move_data.go                 Data directory relocation
    upload_size.go               PHP upload limit
    ssl.go                       Self-signed SSL + HTTPS redirect
    port_forward.go              Trusted domains + router guidance
    complete.go                  Scrollable summary with viewport
```

### Design Principles

1. **Each step is a state machine.** Phases flow forward on success, backward on Escape, and sideways on error (retry/skip). No step can leave the system in a broken state.

2. **View never blocks.** Image downloads are async. Command execution happens in background goroutines. The UI always remains responsive.

3. **Shared state, isolated logic.** Steps communicate through a shared `State` struct but own their internal phase management. Adding a new step means implementing one interface.

4. **Validation at the boundary.** User input is validated before any system command runs. MySQL identifiers, PHP sizes, domain names, and paths all have regex-based validators.

## Contributing

Contributions welcome. The main areas where help is appreciated:

- **Testing on different Pi models** (Pi 3, Pi 4, Pi 5, Pi Zero 2)
- **Theming** for light terminal backgrounds
- **Let's Encrypt integration** as an alternative to self-signed certificates
- **Localization** for non-English users

## Credits

- [Nextcloud](https://nextcloud.com/) — the self-hosted cloud platform
- [Bubbletea](https://github.com/charmbracelet/bubbletea) — terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [PiMyLifeUp](https://pimylifeup.com/raspberry-pi-nextcloud/) — original tutorial this installer is based on

## License

MIT
