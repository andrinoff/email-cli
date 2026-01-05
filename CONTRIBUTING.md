# Contributing to Matcha

First off, thank you for considering contributing to Matcha! It's people like you that make this project great.

## How Can I Contribute?

### Reporting Bugs

If you find a bug, please open an issue on our [GitHub issues page](https://github.com/floatpane/matcha/issues). Please include as much detail as possible, including:

- A clear and descriptive title.
- Steps to reproduce the bug.
- What you expected to happen.
- What actually happened.
- Your operating system and terminal.

### Suggesting Enhancements

If you have an idea for a new feature or an improvement to an existing one, please open an issue on our [GitHub issues page](https://github.com/floatpane/matcha/issues). Please provide a clear description of the enhancement and why you think it would be a good addition.

### Pull Requests

We love pull requests! If you're ready to contribute code, here's how to get started:

1.  Fork the repository.
2.  Create a new branch for your feature or bug fix: `git checkout -b feature/your-feature-name` or `git checkout -b fix/your-bug-fix`.
3.  Make your changes.
4.  Ensure your code is formatted with `go fmt ./...`.
5.  Run tests with `go test ./...`.
6.  Commit your changes with a descriptive commit message.
7.  Push your branch to your fork.
8.  Open a pull request to the `master` branch of the main repository.

## Development Setup

To get started with development, you'll need to have Go installed.

1.  Clone the repository:
    ```bash
    git clone https://github.com/floatpane/matcha.git
    cd matcha
    ```
2.  Install dependencies:
    ```bash
    go mod tidy
    ```
3.  Build the project:
    ```bash
    go build -trimpath -ldflags="-s -w" -o matcha
    ```
4.  Run the application:
    ```bash
    ./matcha
    ```

## Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms. Please read the [Code of Conduct](CODE_OF_CONDUCT.md).