package view

import (
	"bytes"
	"fmt"
	"io"
	"mime/quotedprintable"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

// hyperlink formats a string as a terminal-clickable hyperlink.
func hyperlink(url, text string) string {
	if text == "" {
		text = url
	}
	return fmt.Sprintf("\x1b]8;;%s\x07%s\x1b]8;;\x07", url, text)
}

func decodeQuotedPrintable(s string) (string, error) {
	reader := quotedprintable.NewReader(strings.NewReader(s))
	body, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// markdownToHTML converts a Markdown string to an HTML string.
func markdownToHTML(md []byte) []byte {
	var buf bytes.Buffer
	p := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML in email.
		),
	)
	if err := p.Convert(md, &buf); err != nil {
		return md // Fallback to original markdown.
	}
	return buf.Bytes()
}

// ProcessBody takes a raw email body, decodes it, and formats it as plain
// text with terminal hyperlinks.
func ProcessBody(rawBody string) (string, error) {
	decodedBody, err := decodeQuotedPrintable(rawBody)
	if err != nil {
		// If decoding fails, fallback to the raw body
		decodedBody = rawBody
	}

	// Convert markdown to HTML before processing
	htmlBody := markdownToHTML([]byte(decodedBody))

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBody))
	if err != nil {
		return "", fmt.Errorf("could not parse email body: %w", err)
	}

	// Remove style and script tags to clean up the view
	doc.Find("style, script").Remove()

	// Add newlines after block elements for better spacing
	doc.Find("p, div, h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		s.After("\n\n")
	})

	// Replace <br> tags with newlines
	doc.Find("br").Each(func(i int, s *goquery.Selection) {
		s.ReplaceWithHtml("\n")
	})

	// Format links into terminal hyperlinks
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		s.ReplaceWithHtml(hyperlink(href, s.Text()))
	})

	// Format images into terminal hyperlinks
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}
		alt, _ := s.Attr("alt")

		if alt == "" {
			alt = "Does not contain alt text"
		}
		s.ReplaceWithHtml(hyperlink(src, fmt.Sprintf("\n [Click here to view image: %s] \n", alt)))
	})

	// Get the document's text content, which now includes our formatting
	text := doc.Text()

	// Collapse more than 2 consecutive newlines into 2
	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	return text, nil
}