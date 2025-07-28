# Email CLI 📧

[![Go CI](https://github.com/andrinoff/email-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/andrinoff/email-cli/actions/workflows/ci.yml)
[![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/andrinoff/email-cli)](https://github.com/andrinoff/email-cli/issues)


A beautiful and functional email client for your terminal, built with Go and the charming Bubble Tea TUI library. Never leave your command line to check your inbox or send an email again!

![Main Menu Screenshot](assets/preview.png)

## Features ✨

- **View Your Inbox**: Fetches and displays a list of your most recent emails.
- **Read Emails**: Select an email from your inbox to view its content.
- **Compose and Send**: A simple and intuitive interface for writing and sending new emails.
- **Beautiful TUI**: A clean and modern terminal user interface that's a pleasure to use.
- **Secure**: Uses a local configuration file to store your credentials securely.

## Getting Started 🚀

Follow these instructions to get the email client up and running on your local machine.

### Prerequisites

- **Go**: You need to have Go (version 1.18 or newer) installed on your system. You can download it from the [official Go website](https://golang.org/dl/).

### Installation & Setup

1.  **Clone the repository:**

    ```bash
    git clone [https://github.com/andrinoff/email-cli.git](https://github.com/andrinoff/email-cli.git)
    cd email-cli
    ```

2.  Login

    You will have 4 input fields

    1. Provider (currently, `gmail` or `icloud`)
    2. Email
    3. App-specific Password (TODO: add a guide)
    4. Name to sign off (OPTIONAL, but recommended)

3.  **Run the application:**
    Once your `config.json` file is set up, you can run the application directly from your terminal:
    ```bash
    go run .
    ```

## Usage ⌨️

The Email CLI is designed to be intuitive and easy to navigate with your keyboard.

- **Main Menu**: Use the `Up`/`Down` arrow keys or `k`/`j` to navigate the main menu choices. Press `Enter` to select an option.
- **Inbox View**: In the inbox, use the arrow keys to scroll through your emails. Press `Enter` to open and view a selected email.
- **Composer**:
  - Use `Tab` or the arrow keys to move between the "To", "Subject", and "Body" fields.
  - When you are finished composing your email, navigate to the "Body" field and press `Enter` to send it.
- **Go Back/Quit**:
  - Press `Esc` from any view to return to the main menu.
  - Press `Ctrl+C` at any time to quit the application.

## Built With 🛠️

This project is built with some fantastic Go libraries:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - A powerful TUI (Text User Interface) framework.
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - A library for beautiful, style-driven layouts in the terminal.
- [Bubbles](https://github.com/charmbracelet/bubbles) - A collection of ready-to-use components for Bubble Tea applications.

## License 📄

This project is distributed under the MIT License. See the `LICENSE` file for more information.

---

Made with ❤️ by [@andrinoff](https://andrinoff.com)
