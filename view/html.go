package view

import (
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

// ProcessBody takes a raw email body, decodes it, and formats it as plain
// text with terminal hyperlinks.
func ProcessBody(rawBody string) (string, error) {
	decodedBody, err := decodeQuotedPrintable(rawBody)
	if err != nil {
		// If decoding fails, fallback to the raw body
		decodedBody = rawBody
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(decodedBody))
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

	// Return the document's text content, which now includes our formatting
	return doc.Text(), nil
}