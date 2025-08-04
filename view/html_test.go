package view

import (
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
			expected: "[Click here to view image: alt text]",
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
			input:    `![alt text](http://example.com/img.png)>`,
			expected: "[Click here to view image: alt text]",
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
