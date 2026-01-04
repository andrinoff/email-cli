package view

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

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

func TestHyperlink(t *testing.T) {
	testCases := []struct {
		name               string
		url                string
		text               string
		hyperlinkSupported bool
		expected           string
	}{
		{
			name:               "Supported terminal with different text",
			url:                "https://example.com",
			text:               "Click here",
			hyperlinkSupported: true,
			expected:           "\x1b]8;;https://example.com\x07Click here\x1b]8;;\x07",
		},
		{
			name:               "Supported terminal with same text as URL",
			url:                "https://example.com",
			text:               "https://example.com",
			hyperlinkSupported: true,
			expected:           "\x1b]8;;https://example.com\x07https://example.com\x1b]8;;\x07",
		},
		{
			name:               "Supported terminal with empty text",
			url:                "https://example.com",
			text:               "",
			hyperlinkSupported: true,
			expected:           "\x1b]8;;https://example.com\x07https://example.com\x1b]8;;\x07",
		},
		{
			name:               "Unsupported terminal with different text",
			url:                "https://example.com",
			text:               "Click here",
			hyperlinkSupported: false,
			expected:           "Click here &lt;https://example.com&gt;",
		},
		{
			name:               "Unsupported terminal with same text as URL",
			url:                "https://example.com",
			text:               "https://example.com",
			hyperlinkSupported: false,
			expected:           "&lt;https://example.com&gt;",
		},
		{
			name:               "Unsupported terminal with empty text",
			url:                "https://example.com",
			text:               "",
			hyperlinkSupported: false,
			expected:           "&lt;https://example.com&gt;",
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

			result := hyperlink(tc.url, tc.text)

			// Restore original env vars
			os.Setenv("TERM", origTerm)
			os.Setenv("TERM_PROGRAM", origTermProgram)
			os.Setenv("VTE_VERSION", origVteVersion)
			os.Setenv("KITTY_WINDOW_ID", origKittyWindowID)
			os.Setenv("GHOSTTY_RESOURCES_DIR", origGhosttyResources)
			os.Setenv("WEZTERM_EXECUTABLE", origWeztermExecutable)

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
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
			name:     "With link HTML",
			input:    `<a href="http://example.com">Click here</a>`,
			expected: "Click here",
		},
		{
			name:     "With image HTML",
			input:    `<img src="http://example.com/img.png" alt="alt text">`,
			expected: "[Image: alt text, http://example.com/img.png]",
		},
		{
			name:     "With headers HTML",
			input:    "<h1>Header 1</h1>",
			expected: "Header 1",
		},
		{
			name:     "With link Markdown",
			input:    `[Click here](http://example.com)`,
			expected: "Click here",
		},
		{
			name:     "With image Markdown",
			input:    `![alt text](http://example.com/img.png)`,
			expected: "[Image: alt text, http://example.com/img.png]",
		},
		{
			name:     "With headers Markdown",
			input:    "# Header 1",
			expected: "Header 1",
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
