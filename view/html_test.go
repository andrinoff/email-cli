package view

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// clearAllTerminalEnv clears all environment variables that could indicate terminal capabilities
func clearAllTerminalEnv() {
	// Clear hyperlink support indicators
	os.Unsetenv("VTE_VERSION")
	os.Unsetenv("KITTY_WINDOW_ID")
	os.Unsetenv("GHOSTTY_RESOURCES_DIR")
	os.Unsetenv("WEZTERM_EXECUTABLE")
	os.Unsetenv("WEZTERM_CONFIG_FILE")
	os.Unsetenv("ITERM_SESSION_ID")
	os.Unsetenv("ITERM_PROFILE")
	os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
	os.Unsetenv("WARP_COMBINED_PROMPT_COMMAND_FINISHED")
	os.Unsetenv("KONSOLE_DBUS_SESSION")
	os.Unsetenv("KONSOLE_VERSION")

	// Set basic terminal that doesn't support anything special
	os.Setenv("TERM", "xterm")
	os.Setenv("TERM_PROGRAM", "basic")
}

func TestDecodeQuotedPrintable(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple case",
			input:    "Hello=2C world=21",
			expected: "Hello, world!",
		},
		{
			name:     "With soft line break",
			input:    "This is a long line that gets wrapped=\r\n and continues here.",
			expected: "This is a long line that gets wrapped and continues here.",
		},
		{
			name:     "No encoding",
			input:    "Just a plain string.",
			expected: "Just a plain string.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := decodeQuotedPrintable(tc.input)
			if err != nil {
				t.Fatalf("decodeQuotedPrintable() failed: %v", err)
			}
			if decoded != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, decoded)
			}
		})
	}
}

func TestMarkdownToHTML(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Heading",
			input:    "# Hello",
			expected: "<h1>Hello</h1>",
		},
		{
			name:     "Bold",
			input:    "**bold text**",
			expected: "<p><strong>bold text</strong></p>",
		},
		{
			name:     "Link",
			input:    "[link](http://example.com)",
			expected: `<p><a href="http://example.com">link</a></p>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			html := markdownToHTML([]byte(tc.input))
			// Trim newlines for consistent comparison
			if strings.TrimSpace(string(html)) != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, html)
			}
		})
	}
}

func TestHyperlinkSupported(t *testing.T) {
	testCases := []struct {
		name              string
		term              string
		termProgram       string
		vteVersion        string
		kittyWindowID     string
		ghosttyResources  string
		weztermExecutable string
		expected          bool
	}{
		{
			name:     "Kitty terminal",
			term:     "xterm-kitty",
			expected: true,
		},
		{
			name:     "Ghostty terminal",
			term:     "ghostty",
			expected: true,
		},
		{
			name:     "WezTerm terminal",
			term:     "wezterm",
			expected: true,
		},
		{
			name:     "Alacritty terminal",
			term:     "alacritty",
			expected: true,
		},
		{
			name:        "iTerm2 via TERM_PROGRAM",
			term:        "xterm-256color",
			termProgram: "iTerm.app",
			expected:    true,
		},
		{
			name:       "VTE-based terminal",
			term:       "xterm-256color",
			vteVersion: "0.64.2",
			expected:   true,
		},
		{
			name:          "Kitty via env var",
			term:          "xterm-256color",
			kittyWindowID: "1",
			expected:      true,
		},
		{
			name:             "Ghostty via env var",
			term:             "xterm-256color",
			ghosttyResources: "/opt/ghostty",
			expected:         true,
		},
		{
			name:              "WezTerm via env var",
			term:              "xterm-256color",
			weztermExecutable: "/usr/bin/wezterm",
			expected:          true,
		},
		{
			name:     "Unsupported terminal",
			term:     "xterm",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original env vars
			origTerm := os.Getenv("TERM")
			origTermProgram := os.Getenv("TERM_PROGRAM")
			origVteVersion := os.Getenv("VTE_VERSION")
			origKittyWindowID := os.Getenv("KITTY_WINDOW_ID")
			origGhosttyResources := os.Getenv("GHOSTTY_RESOURCES_DIR")
			origWeztermExecutable := os.Getenv("WEZTERM_EXECUTABLE")

			// Set test env vars
			os.Setenv("TERM", tc.term)
			os.Setenv("TERM_PROGRAM", tc.termProgram)
			os.Setenv("VTE_VERSION", tc.vteVersion)
			os.Setenv("KITTY_WINDOW_ID", tc.kittyWindowID)
			os.Setenv("GHOSTTY_RESOURCES_DIR", tc.ghosttyResources)
			os.Setenv("WEZTERM_EXECUTABLE", tc.weztermExecutable)

			// Test the function
			result := hyperlinkSupported()

			// Restore original env vars
			os.Setenv("TERM", origTerm)
			os.Setenv("TERM_PROGRAM", origTermProgram)
			os.Setenv("VTE_VERSION", origVteVersion)
			os.Setenv("KITTY_WINDOW_ID", origKittyWindowID)
			os.Setenv("GHOSTTY_RESOURCES_DIR", origGhosttyResources)
			os.Setenv("WEZTERM_EXECUTABLE", origWeztermExecutable)

			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestImageProtocolSupported(t *testing.T) {
	// Save original environment variables
	origTerm := os.Getenv("TERM")
	origKittyWindow := os.Getenv("KITTY_WINDOW_ID")
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origGhosttyResources := os.Getenv("GHOSTTY_RESOURCES_DIR")
	origItermlSession := os.Getenv("ITERM_SESSION_ID")
	origWeztermExec := os.Getenv("WEZTERM_EXECUTABLE")
	origWarpLocal := os.Getenv("WARP_IS_LOCAL_SHELL_SESSION")
	origKonsoleDBus := os.Getenv("KONSOLE_DBUS_SESSION")

	// Restore environment variables after test
	defer func() {
		os.Setenv("TERM", origTerm)
		os.Setenv("KITTY_WINDOW_ID", origKittyWindow)
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("GHOSTTY_RESOURCES_DIR", origGhosttyResources)
		os.Setenv("ITERM_SESSION_ID", origItermlSession)
		os.Setenv("WEZTERM_EXECUTABLE", origWeztermExec)
		os.Setenv("WARP_IS_LOCAL_SHELL_SESSION", origWarpLocal)
		os.Setenv("KONSOLE_DBUS_SESSION", origKonsoleDBus)
	}()

	testCases := []struct {
		name        string
		setupEnv    func()
		clearAllEnv func()
		expected    bool
	}{
		{
			name: "No supported terminals",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("TERM_PROGRAM", "basic")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
				os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
				os.Unsetenv("KONSOLE_DBUS_SESSION")
			},
			expected: false,
		},
		{
			name: "Kitty supported via TERM",
			setupEnv: func() {
				os.Setenv("TERM", "xterm-kitty")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
				os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
				os.Unsetenv("KONSOLE_DBUS_SESSION")
			},
			expected: true,
		},
		{
			name: "Kitty supported via KITTY_WINDOW_ID",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("KITTY_WINDOW_ID", "1")
			},
			clearAllEnv: func() {
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
				os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
				os.Unsetenv("KONSOLE_DBUS_SESSION")
			},
			expected: true,
		},
		{
			name: "Ghostty supported via TERM_PROGRAM",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("TERM_PROGRAM", "ghostty")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
				os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
				os.Unsetenv("KONSOLE_DBUS_SESSION")
			},
			expected: true,
		},
		{
			name: "iTerm2 supported via TERM_PROGRAM",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("TERM_PROGRAM", "iterm.app")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
				os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
				os.Unsetenv("KONSOLE_DBUS_SESSION")
			},
			expected: true,
		},
		{
			name: "WezTerm supported via WEZTERM_EXECUTABLE",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("WEZTERM_EXECUTABLE", "/usr/bin/wezterm")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
				os.Unsetenv("KONSOLE_DBUS_SESSION")
			},
			expected: true,
		},
		{
			name: "Warp supported via WARP_IS_LOCAL_SHELL_SESSION",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("WARP_IS_LOCAL_SHELL_SESSION", "1")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
				os.Unsetenv("KONSOLE_DBUS_SESSION")
			},
			expected: true,
		},
		{
			name: "Konsole supported via KONSOLE_DBUS_SESSION",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("KONSOLE_DBUS_SESSION", "/Sessions/1")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
				os.Unsetenv("WARP_IS_LOCAL_SHELL_SESSION")
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.clearAllEnv()
			tc.setupEnv()

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestHyperlinkSupported(t *testing.T) {
	// Save original environment variables
	origTerm := os.Getenv("TERM")
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origVTEVersion := os.Getenv("VTE_VERSION")
	origKittyWindow := os.Getenv("KITTY_WINDOW_ID")
	origGhosttyResources := os.Getenv("GHOSTTY_RESOURCES_DIR")
	origWeztermExec := os.Getenv("WEZTERM_EXECUTABLE")

	// Restore environment variables after test
	defer func() {
		os.Setenv("TERM", origTerm)
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("VTE_VERSION", origVTEVersion)
		os.Setenv("KITTY_WINDOW_ID", origKittyWindow)
		os.Setenv("GHOSTTY_RESOURCES_DIR", origGhosttyResources)
		os.Setenv("WEZTERM_EXECUTABLE", origWeztermExec)
	}()

	testCases := []struct {
		name        string
		setupEnv    func()
		clearAllEnv func()
		expected    bool
	}{
		{
			name: "No hyperlink support",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("TERM_PROGRAM", "basic")
			},
			clearAllEnv: func() {
				os.Unsetenv("VTE_VERSION")
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			},
			expected: false,
		},
		{
			name: "Kitty hyperlink support via TERM",
			setupEnv: func() {
				os.Setenv("TERM", "xterm-kitty")
			},
			clearAllEnv: func() {
				os.Unsetenv("VTE_VERSION")
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			},
			expected: true,
		},
		{
			name: "VTE-based terminal hyperlink support",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("VTE_VERSION", "0.60.3")
			},
			clearAllEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			},
			expected: true,
		},
		{
			name: "iTerm2 hyperlink support",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("TERM_PROGRAM", "iterm.app")
			},
			clearAllEnv: func() {
				os.Unsetenv("VTE_VERSION")
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			},
			expected: true,
		},
		{
			name: "WezTerm hyperlink support",
			setupEnv: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("WEZTERM_EXECUTABLE", "/usr/bin/wezterm")
			},
			clearAllEnv: func() {
				os.Unsetenv("VTE_VERSION")
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.clearAllEnv()
			tc.setupEnv()

			result := hyperlinkSupported()
			if result != tc.expected {
				t.Errorf("Expected %t, got %t", tc.expected, result)
			}
		})
	}
}

func TestProcessBodyWithHyperlinkSupport(t *testing.T) {
	// Save original environment variables
	origTerm := os.Getenv("TERM")
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origVTEVersion := os.Getenv("VTE_VERSION")
	origKittyWindow := os.Getenv("KITTY_WINDOW_ID")

	// Restore environment variables after test
	defer func() {
		os.Setenv("TERM", origTerm)
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("VTE_VERSION", origVTEVersion)
		os.Setenv("KITTY_WINDOW_ID", origKittyWindow)
	}()

	h1Style := lipgloss.NewStyle().SetString("H1")
	h2Style := lipgloss.NewStyle().SetString("H2")
	bodyStyle := lipgloss.NewStyle().SetString("BODY")

	testCases := []struct {
		name                string
		setupHyperlinks     func()
		input               string
		expectedContains    string
		expectedNotContains string
	}{
		{
			name: "Link with hyperlink support",
			setupHyperlinks: func() {
				os.Setenv("TERM", "xterm-kitty")
				os.Unsetenv("VTE_VERSION")
				os.Unsetenv("KITTY_WINDOW_ID")
			},
			input:               `<a href="http://example.com">Click here</a>`,
			expectedContains:    "Click here",
			expectedNotContains: "&lt;http://example.com&gt;",
		},
		{
			name: "Link without hyperlink support",
			setupHyperlinks: func() {
				clearAllTerminalEnv()
			},
			input:            `<a href="http://example.com">Click here</a>`,
			expectedContains: "Click here <http://example.com>",
		},
		{
			name: "Image link with hyperlink support",
			setupHyperlinks: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("VTE_VERSION", "0.60.3")
				os.Unsetenv("KITTY_WINDOW_ID")
			},
			input:               `<img src="http://example.com/img.png" alt="alt text">`,
			expectedContains:    "[Click here to view image: alt text]",
			expectedNotContains: "&lt;http://example.com/img.png&gt;",
		},
		{
			name: "Image link without hyperlink support",
			setupHyperlinks: func() {
				clearAllTerminalEnv()
			},
			input:            `<img src="http://example.com/img.png" alt="alt text">`,
			expectedContains: "[Image: alt text, http://example.com/img.png]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupHyperlinks()

			processed, err := ProcessBody(tc.input, h1Style, h2Style, bodyStyle)
			if err != nil {
				t.Fatalf("ProcessBody() failed: %v", err)
			}

			if !strings.Contains(processed, tc.expectedContains) {
				t.Errorf("Processed body does not contain expected text.\nGot: %q\nWant to contain: %q", processed, tc.expectedContains)
			}

			if tc.expectedNotContains != "" && strings.Contains(processed, tc.expectedNotContains) {
				t.Errorf("Processed body contains unexpected text.\nGot: %q\nShould not contain: %q", processed, tc.expectedNotContains)
			}
		})
	}
}

func TestProcessBodyWithImageProtocol(t *testing.T) {
	// Save original environment variables
	origTerm := os.Getenv("TERM")
	origTermProgram := os.Getenv("TERM_PROGRAM")
	origKittyWindow := os.Getenv("KITTY_WINDOW_ID")
	origGhosttyResources := os.Getenv("GHOSTTY_RESOURCES_DIR")
	origItermlSession := os.Getenv("ITERM_SESSION_ID")
	origWeztermExec := os.Getenv("WEZTERM_EXECUTABLE")

	// Restore environment variables after test
	defer func() {
		os.Setenv("TERM", origTerm)
		os.Setenv("TERM_PROGRAM", origTermProgram)
		os.Setenv("KITTY_WINDOW_ID", origKittyWindow)
		os.Setenv("GHOSTTY_RESOURCES_DIR", origGhosttyResources)
		os.Setenv("ITERM_SESSION_ID", origItermlSession)
		os.Setenv("WEZTERM_EXECUTABLE", origWeztermExec)
	}()

	h1Style := lipgloss.NewStyle().SetString("H1")
	h2Style := lipgloss.NewStyle().SetString("H2")
	bodyStyle := lipgloss.NewStyle().SetString("BODY")

	// Create a simple base64 PNG image (1x1 pixel white PNG)
	testBase64PNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="

	testCases := []struct {
		name                string
		setupImageProtocol  func()
		clearAllImageEnv    func()
		input               string
		expectedContains    string
		expectedNotContains string
	}{
		{
			name: "Data URI image with Kitty support",
			setupImageProtocol: func() {
				os.Setenv("TERM", "xterm-kitty")
			},
			clearAllImageEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			},
			input:               `<img src="data:image/png;base64,` + testBase64PNG + `" alt="test image">`,
			expectedContains:    "\x1b_Gf=100,a=T,q=2,C=1,m=0;",
			expectedNotContains: "[Image: test image,",
		},
		{
			name: "Data URI image with iTerm2 support",
			setupImageProtocol: func() {
				os.Setenv("TERM", "xterm")
				os.Setenv("TERM_PROGRAM", "iterm.app")
			},
			clearAllImageEnv: func() {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("ITERM_SESSION_ID")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			},
			input:               `<img src="data:image/png;base64,` + testBase64PNG + `" alt="test image">`,
			expectedContains:    "\x1b]1337;File=inline=1:",
			expectedNotContains: "[Image: test image,",
		},
		{
			name: "Data URI image without protocol support",
			setupImageProtocol: func() {
				clearAllTerminalEnv()
			},
			clearAllImageEnv: func() {
				// This is handled by clearAllTerminalEnv now
			},
			input:            `<img src="data:image/png;base64,` + testBase64PNG + `" alt="test image">`,
			expectedContains: "[Image: test image,",
		},
		{
			name: "Remote image with WezTerm support (has hyperlink support)",
			setupImageProtocol: func() {
				clearAllTerminalEnv()
				os.Setenv("WEZTERM_EXECUTABLE", "/usr/bin/wezterm")
			},
			clearAllImageEnv: func() {
				// This is handled by clearAllTerminalEnv now
			},
			input:            `<img src="http://example.com/img.png" alt="remote image">`,
			expectedContains: "[Click here to view image: remote image]", // Remote images won't render without actual fetch, but hyperlinks work
		},
		{
			name: "Remote image without protocol support",
			setupImageProtocol: func() {
				clearAllTerminalEnv()
			},
			clearAllImageEnv: func() {
				// This is handled by clearAllTerminalEnv now
			},
			input:            `<img src="http://example.com/img.png" alt="remote image">`,
			expectedContains: "[Image: remote image,",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.clearAllImageEnv()
			tc.setupImageProtocol()

			processed, err := ProcessBody(tc.input, h1Style, h2Style, bodyStyle)
			if err != nil {
				t.Fatalf("ProcessBody() failed: %v", err)
			}

			if !strings.Contains(processed, tc.expectedContains) {
				t.Errorf("Processed body does not contain expected text.\nGot: %q\nWant to contain: %q", processed, tc.expectedContains)
			}

			if tc.expectedNotContains != "" && strings.Contains(processed, tc.expectedNotContains) {
				t.Errorf("Processed body contains unexpected text.\nGot: %q\nShould not contain: %q", processed, tc.expectedNotContains)
			}
		})
	}
}

func TestProcessBody(t *testing.T) {
	h1Style := lipgloss.NewStyle().SetString("H1")
	h2Style := lipgloss.NewStyle().SetString("H2")
	bodyStyle := lipgloss.NewStyle().SetString("BODY")

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple HTML",
			input:    "<p>Hello, world!</p>",
			expected: "Hello, world!",
		},
		{
			name:     "With headers HTML",
			input:    "<h1>Header 1</h1>",
			expected: "Header 1",
		},
		{
			name:     "With headers Markdown",
			input:    "# Header 1",
			expected: "Header 1",
		},
		{
			name:     "Plain text",
			input:    "Just plain text without any markup",
			expected: "Just plain text without any markup",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processed, err := ProcessBody(tc.input, h1Style, h2Style, bodyStyle)
			if err != nil {
				t.Fatalf("ProcessBody() failed: %v", err)
			}
			// Use Contains because styles add ANSI codes
			if !strings.Contains(processed, tc.expected) {
				t.Errorf("Processed body does not contain expected text.\nGot: %q\nWant to contain: %q", processed, tc.expected)
			}
		})
	}
}

func TestProcessBodyWithHyperlinkSupport(t *testing.T) {
	h1Style := lipgloss.NewStyle()
	h2Style := lipgloss.NewStyle()
	bodyStyle := lipgloss.NewStyle()

	testCases := []struct {
		name                string
		input               string
		hyperlinkSupported  bool
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:                "Link with hyperlink support",
			input:               `<a href="https://example.com">Click here</a>`,
			hyperlinkSupported:  true,
			expectedContains:    []string{"\x1b]8;;https://example.com\x07Click here\x1b]8;;\x07"},
			expectedNotContains: []string{"<https://example.com>"},
		},
		{
			name:                "Link without hyperlink support",
			input:               `<a href="https://example.com">Click here</a>`,
			hyperlinkSupported:  false,
			expectedContains:    []string{"Click here <https://example.com>"},
			expectedNotContains: []string{"\x1b]8;;"},
		},
		{
			name:                "Image with hyperlink support",
			input:               `<img src="https://example.com/image.png" alt="Test image">`,
			hyperlinkSupported:  true,
			expectedContains:    []string{"\x1b]8;;https://example.com/image.png\x07", "[Click here to view image: Test image]"},
			expectedNotContains: []string{"<https://example.com/image.png>"},
		},
		{
			name:                "Image without hyperlink support",
			input:               `<img src="https://example.com/image.png" alt="Test image">`,
			hyperlinkSupported:  false,
			expectedContains:    []string{"[Image: Test image, https://example.com/image.png]"},
			expectedNotContains: []string{"\x1b]8;;", "[Click here to view image:"},
		},
		{
			name:                "Markdown link with hyperlink support",
			input:               `[Visit our site](https://example.com)`,
			hyperlinkSupported:  true,
			expectedContains:    []string{"\x1b]8;;https://example.com\x07Visit our site\x1b]8;;\x07"},
			expectedNotContains: []string{"<https://example.com>"},
		},
		{
			name:                "Markdown link without hyperlink support",
			input:               `[Visit our site](https://example.com)`,
			hyperlinkSupported:  false,
			expectedContains:    []string{"Visit our site <https://example.com>"},
			expectedNotContains: []string{"\x1b]8;;"},
		},
		{
			name:                "Markdown image with hyperlink support",
			input:               `![Beautiful sunset](https://example.com/sunset.jpg)`,
			hyperlinkSupported:  true,
			expectedContains:    []string{"\x1b]8;;https://example.com/sunset.jpg\x07", "[Click here to view image: Beautiful sunset]"},
			expectedNotContains: []string{"<https://example.com/sunset.jpg>"},
		},
		{
			name:                "Markdown image without hyperlink support",
			input:               `![Beautiful sunset](https://example.com/sunset.jpg)`,
			hyperlinkSupported:  false,
			expectedContains:    []string{"[Image: Beautiful sunset, https://example.com/sunset.jpg]"},
			expectedNotContains: []string{"\x1b]8;;", "[Click here to view image:"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original env vars
			origTerm := os.Getenv("TERM")
			origTermProgram := os.Getenv("TERM_PROGRAM")
			origVteVersion := os.Getenv("VTE_VERSION")
			origKittyWindowID := os.Getenv("KITTY_WINDOW_ID")
			origGhosttyResources := os.Getenv("GHOSTTY_RESOURCES_DIR")
			origWeztermExecutable := os.Getenv("WEZTERM_EXECUTABLE")

			// Set env to control hyperlink support
			if tc.hyperlinkSupported {
				os.Setenv("TERM", "xterm-kitty")
			} else {
				// Clear all environment variables that could indicate hyperlink support
				os.Setenv("TERM", "xterm")
				os.Unsetenv("TERM_PROGRAM")
				os.Unsetenv("VTE_VERSION")
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			}

			processed, err := ProcessBody(tc.input, h1Style, h2Style, bodyStyle)

			// Restore original env vars
			os.Setenv("TERM", origTerm)
			os.Setenv("TERM_PROGRAM", origTermProgram)
			os.Setenv("VTE_VERSION", origVteVersion)
			os.Setenv("KITTY_WINDOW_ID", origKittyWindowID)
			os.Setenv("GHOSTTY_RESOURCES_DIR", origGhosttyResources)
			os.Setenv("WEZTERM_EXECUTABLE", origWeztermExecutable)

			if err != nil {
				t.Fatalf("ProcessBody() failed: %v", err)
			}

			// Check that expected strings are present
			for _, expected := range tc.expectedContains {
				if !strings.Contains(processed, expected) {
					t.Errorf("Expected processed body to contain %q, but it didn't.\nProcessed: %q", expected, processed)
				}
			}

			// Check that unexpected strings are not present
			for _, notExpected := range tc.expectedNotContains {
				if strings.Contains(processed, notExpected) {
					t.Errorf("Expected processed body to NOT contain %q, but it did.\nProcessed: %q", notExpected, processed)
				}
			}
		})
	}
}

func TestImageAndLinkFallbackBehavior(t *testing.T) {
	h1Style := lipgloss.NewStyle()
	h2Style := lipgloss.NewStyle()
	bodyStyle := lipgloss.NewStyle()

	testCases := []struct {
		name                   string
		input                  string
		hyperlinkSupported     bool
		imageProtocolSupported bool
		expectedContains       []string
		expectedNotContains    []string
	}{
		{
			name:                   "Full support - both image protocol and hyperlinks",
			input:                  `<img src="https://example.com/image.png" alt="Test image"> and <a href="https://example.com">link</a>`,
			hyperlinkSupported:     true,
			imageProtocolSupported: true,
			expectedContains:       []string{"\x1b]8;;https://example.com\x07link\x1b]8;;\x07", "\x1b]8;;https://example.com/image.png\x07", "[Click here to view image: Test image]"},
			expectedNotContains:    []string{"<https://example.com>", "[Image:"},
		},
		{
			name:                   "Hyperlink support only - no image protocol",
			input:                  `<img src="https://example.com/image.png" alt="Test image"> and <a href="https://example.com">link</a>`,
			hyperlinkSupported:     true,
			imageProtocolSupported: false,
			expectedContains:       []string{"\x1b]8;;https://example.com\x07link\x1b]8;;\x07", "\x1b]8;;https://example.com/image.png\x07", "[Click here to view image: Test image]"},
			expectedNotContains:    []string{"<https://example.com>", "[Image:"},
		},
		{
			name:                   "No support - plain text fallback",
			input:                  `<img src="https://example.com/image.png" alt="Test image"> and <a href="https://example.com">link</a>`,
			hyperlinkSupported:     false,
			imageProtocolSupported: false,
			expectedContains:       []string{"link <https://example.com>", "[Image: Test image, https://example.com/image.png]"},
			expectedNotContains:    []string{"\x1b]8;;", "[Click here to view image:"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original env vars
			origTerm := os.Getenv("TERM")
			origTermProgram := os.Getenv("TERM_PROGRAM")
			origVteVersion := os.Getenv("VTE_VERSION")
			origKittyWindowID := os.Getenv("KITTY_WINDOW_ID")
			origGhosttyResources := os.Getenv("GHOSTTY_RESOURCES_DIR")
			origWeztermExecutable := os.Getenv("WEZTERM_EXECUTABLE")

			// Set environment for hyperlink support
			if tc.hyperlinkSupported {
				os.Setenv("TERM", "xterm-kitty")
			} else {
				os.Setenv("TERM", "xterm")
				os.Unsetenv("TERM_PROGRAM")
				os.Unsetenv("VTE_VERSION")
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
				os.Unsetenv("WEZTERM_EXECUTABLE")
			}

			// Set environment for image protocol support
			if tc.imageProtocolSupported && !tc.hyperlinkSupported {
				// This case shouldn't happen in practice, but test it anyway
				os.Setenv("KITTY_WINDOW_ID", "1")
			} else if !tc.imageProtocolSupported {
				os.Unsetenv("KITTY_WINDOW_ID")
				os.Unsetenv("GHOSTTY_RESOURCES_DIR")
			}

			processed, err := ProcessBody(tc.input, h1Style, h2Style, bodyStyle)

			// Restore original env vars
			os.Setenv("TERM", origTerm)
			os.Setenv("TERM_PROGRAM", origTermProgram)
			os.Setenv("VTE_VERSION", origVteVersion)
			os.Setenv("KITTY_WINDOW_ID", origKittyWindowID)
			os.Setenv("GHOSTTY_RESOURCES_DIR", origGhosttyResources)
			os.Setenv("WEZTERM_EXECUTABLE", origWeztermExecutable)

			if err != nil {
				t.Fatalf("ProcessBody() failed: %v", err)
			}

			// Check that expected strings are present
			for _, expected := range tc.expectedContains {
				if !strings.Contains(processed, expected) {
					t.Errorf("Expected processed body to contain %q, but it didn't.\nProcessed: %q", expected, processed)
				}
			}

			// Check that unexpected strings are not present
			for _, notExpected := range tc.expectedNotContains {
				if strings.Contains(processed, notExpected) {
					t.Errorf("Expected processed body to NOT contain %q, but it did.\nProcessed: %q", notExpected, processed)
				}
			}
		})
	}
}
