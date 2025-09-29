# LIFX Dash

A simple, dark-theme friendly dashboard for controlling LIFX devices on your local network, built in Go using [Fyne](https://fyne.io/) and [lifxlan-go](https://github.com/alessio-palumbo/lifxlan-go).

---

## Features

- Discover LIFX devices on your LAN.
- Toggle devices on/off.
- Visual status indicator with color for each device.
- Group devices with collapsible sections.
- Simple and responsive grid layout.
- Cross-platform: Windows, macOS, Linux.

---

## Download

Download the latest release for your operating system from the [GitHub Releases page](https://github.com/alessio-palumbo/lifx-dash/releases).

| OS      | File                          |
| ------- | ----------------------------- |
| macOS   | `lifx-dash-darwin-<arch>.zip` |
| Linux   | `lifx-dash-linux-<arch>.zip`  |
| Windows | `lifx-dash-windows-amd64.zip` |

> Replace `<arch>` with your CPU architecture (e.g., `arm64`, `amd64`).

---

## Installation

1. Unzip the downloaded archive.
2. You will find the application executable and the `README.md` + `LICENSE` inside.

### macOS

- Double-click `lifx-dash.app` to open.
- **Note:** The first time you open the app, macOS may block it due to security settings. To bypass:

```bash
xattr -rd com.apple.quarantine "lifx-dash.app"
```

### Linux

- Run the application executable:

```bash
./usr/bin/lifx-dash
```

### Windows

- Unzip and double-click lifx-dash.exe to run.
- You may see a SmartScreen warning; choose “Run anyway” if prompted.

### Building from Source

If you prefer building from source, you need Go 1.25+ installed:

```bash
go build -o lifx-dash ./cmd/lifx-dash
```
