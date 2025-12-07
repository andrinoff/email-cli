<div align="center">

---

<img src = "assets/logo.png" width=200 height=200>
    
```
                    __       __
    ____ ___  ____ _/ /______/ /_  ____ _
    / __ '__ \/ __ '/ __/ ___/ __ \/ __ '/
    / / / / / / /_/ / /_/ /__/ / / / /_/ /
/_/ /_/ /_/\__,_/\__/\___/_/ /_/\__,_/
```

---

[![Go CI](https://github.com/floatpane/matcha/actions/workflows/ci.yml/badge.svg)](https://github.com/floatpane/matcha/actions/workflows/ci.yml)
[![Go Release](https://github.com/floatpane/matcha/actions/workflows/release.yml/badge.svg)](https://github.com/floatpane/matcha/actions/workflows/release.yml)
[![GoReleaser](https://img.shields.io/badge/GoReleaser-blue?logo=goreleaser)](https://goreleaser.com)
[![Go Version](https://img.shields.io/github/go-mod/go-version/floatpane/matcha)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/floatpane/matcha)](https://goreportcard.com/report/github.com/floatpane/matcha)

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/floatpane/matcha)](https://github.com/floatpane/matcha/releases)
[![GitHub All Releases](https://img.shields.io/github/downloads/floatpane/matcha/total)](https://github.com/floatpane/matcha/releases)
[![GitHub stars](https://img.shields.io/github/stars/floatpane/matcha)](https://github.com/floatpane/matcha/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/floatpane/matcha)](https://github.com/floatpane/matcha/issues)
[![GitHub license](https://img.shields.io/github/license/floatpane/matcha)](https://github.com/floatpane/matcha/blob/master/LICENSE)

[![macOS](https://img.shields.io/badge/macOS-Supported-000000?logo=macos&logoColor=white)](https://www.apple.com/macos)
[![Linux](https://img.shields.io/badge/Linux-Supported-FCC624?logo=linux&logoColor=black)](https://www.linux.org/)
[![Homebrew](https://img.shields.io/badge/homebrew-tap-21648C.svg?logo=homebrew)](https://brew.sh)
[![Snapcraft](https://img.shields.io/badge/snap-available-82BEA0.svg?logo=snapcraft)](https://snapcraft.io/matcha)

[![Patreon](https://img.shields.io/badge/Patreon-F96854?logo=patreon&logoColor=white)](https://patreon.com/andrinoff)
[![GitHub contributors](https://img.shields.io/github/contributors/floatpane/matcha)](https://github.com/floatpane/matcha/graphs/contributors)
[![Built with Bubble Tea](https://img.shields.io/badge/Built%20with-Bubble%20Tea-FF75B7.svg)](https://github.com/charmbracelet/bubbletea)

A beautiful and functional email client for your terminal, built with Go and the charming Bubble Tea TUI library. Never leave your command line to check your inbox or send an email again!


</div>

![Demo GIF](public/assets/demo.gif)

## Features ✨

- **View Your Inbox**: Fetches and displays a list of your most recent emails.
- **Read Emails**: Select an email from your inbox to view its content.
- **Compose and Send**: A simple and intuitive interface for writing and sending new emails.
- **Beautiful TUI**: A clean and modern terminal user interface that's a pleasure to use.
- **Secure**: Uses a local configuration file to store your credentials securely.
- **Supported Providers**: Works with Gmail and iCloud.

## Installation 🚀

There are several ways to install Matcha.

### Package Managers

#### Homebrew 🍺 (macOS & Linux)

```bash
brew tap floatpane/matcha
brew install matcha
```

After installation, run:

```bash
matcha
```

to get started.

### Install using Snap

```bash
sudo snap install matcha
```

### Build from Source 🔨

Matcha is written in **Go**. To build it manually:

1.  Ensure you have Go installed (`go version`).
2.  Clone the repository:

    ```bash
    git clone https://github.com/floatpane/matcha.git
    ```

3.  Navigate to the project folder:

    ```bash
    cd matcha
    ```

4.  Build the binary:

    ```bash
    go build -o matcha
    ```

5.  Run it:
    ```bash
    ./matcha
    ```

## License 📄

This project is distributed under the MIT License. See the `LICENSE` file for more information.
