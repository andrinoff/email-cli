package view

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/quotedprintable"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/lipgloss"
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

func kittySupported() bool {
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "kitty") {
		return true
	}
	return os.Getenv("KITTY_WINDOW_ID") != ""
}

func debugKitty(format string, args ...interface{}) {
	if os.Getenv("DEBUG_KITTY_IMAGES") == "" {
		return
	}
	msg := fmt.Sprintf("[kitty-img] "+format+"\n", args...)
	fmt.Print(msg)
	if path := os.Getenv("DEBUG_KITTY_LOG"); path != "" {
		if f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(msg)
			_ = f.Close()
		}
	}
}

func fetchRemoteBase64(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return ""
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		debugKitty("remote fetch failed url=%s err=%v", url, err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		debugKitty("remote fetch non-200 url=%s status=%d", url, resp.StatusCode)
		return ""
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		debugKitty("remote fetch read error url=%s err=%v", url, err)
		return ""
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		debugKitty("remote decode failed url=%s err=%v", url, err)
		return ""
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		debugKitty("remote png encode failed url=%s err=%v", url, err)
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	debugKitty("remote fetch ok url=%s len=%d", url, len(encoded))
	return encoded
}

func dataURIBase64(uri string) string {
	if !strings.HasPrefix(uri, "data:") {
		return ""
	}
	comma := strings.Index(uri, ",")
	if comma == -1 || comma+1 >= len(uri) {
		return ""
	}
	return uri[comma+1:]
}

func kittyInlineImage(payload string) string {
	if payload == "" {
		return ""
	}

	const chunkSize = 4096
	var b strings.Builder

	// Save cursor before emitting kitty graphics so the terminal position is restored afterward.
	b.WriteString("\x1b[s")

	for offset := 0; offset < len(payload); offset += chunkSize {
		end := offset + chunkSize
		if end > len(payload) {
			end = len(payload)
		}
		more := "0"
		if end < len(payload) {
			more = "1"
		}

		chunk := payload[offset:end]
		if offset == 0 {
			b.WriteString(fmt.Sprintf("\x1b_Gf=100,a=T,q=2,m=%s;%s\x1b\\", more, chunk))
		} else {
			b.WriteString(fmt.Sprintf("\x1b_Gm=%s;%s\x1b\\", more, chunk))
		}
	}

	// Restore cursor and clear any styling so inline images don't displace the UI.
	b.WriteString("\x1b[u\x1b[0m")

	return b.String()
}

type InlineImage struct {
	CID    string
	Base64 string
}

// ProcessBodyWithInline renders the body and resolves CID inline images when provided.
func ProcessBodyWithInline(rawBody string, inline []InlineImage, h1Style, h2Style, bodyStyle lipgloss.Style) (string, error) {
	inlineMap := make(map[string]string, len(inline))
	for _, img := range inline {
		cid := strings.TrimSpace(img.CID)
		cid = strings.TrimPrefix(cid, "<")
		cid = strings.TrimSuffix(cid, ">")
		cid = strings.TrimPrefix(cid, "cid:")
		if cid == "" || img.Base64 == "" {
			continue
		}
		inlineMap[cid] = img.Base64
	}
	return processBody(rawBody, inlineMap, h1Style, h2Style, bodyStyle)
}

// ProcessBody takes a raw email body, decodes it, and formats it as plain
// text with terminal hyperlinks.
func ProcessBody(rawBody string, h1Style, h2Style, bodyStyle lipgloss.Style) (string, error) {
	return processBody(rawBody, nil, h1Style, h2Style, bodyStyle)
}

func processBody(rawBody string, inline map[string]string, h1Style, h2Style, bodyStyle lipgloss.Style) (string, error) {
	decodedBody, err := decodeQuotedPrintable(rawBody)
	if err != nil {
		decodedBody = rawBody
	}

	htmlBody := markdownToHTML([]byte(decodedBody))

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBody))
	if err != nil {
		return "", fmt.Errorf("could not parse email body: %w", err)
	}

	doc.Find("style, script").Remove()

	// Style headers by setting their text content.
	// We use SetText so the h1/h2 tags remain in the document for spacing logic.
	doc.Find("h1").Each(func(i int, s *goquery.Selection) {
		s.SetText(h1Style.Render(s.Text()))
	})

	doc.Find("h2").Each(func(i int, s *goquery.Selection) {
		s.SetText(h2Style.Render(s.Text()))
	})

	// Add newlines after block elements for better spacing.
	doc.Find("p, div, h1, h2").Each(func(i int, s *goquery.Selection) {
		s.After("\n\n")
	})

	// Replace <br> tags with newlines
	doc.Find("br").Each(func(i int, s *goquery.Selection) {
		s.ReplaceWithHtml("\n")
	})

	// Format links and images
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		s.ReplaceWithHtml(hyperlink(href, s.Text()))
	})

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}
		alt, _ := s.Attr("alt")
		if alt == "" {
			alt = "Does not contain alt text"
		}

		if kittySupported() {
			var payload string
			if strings.HasPrefix(src, "data:image/") {
				payload = dataURIBase64(src)
			} else if strings.HasPrefix(src, "cid:") {
				cid := strings.TrimPrefix(src, "cid:")
				cid = strings.Trim(cid, "<>")
				if inline != nil {
					payload = inline[cid]
					debugKitty("cid lookup for %s found=%t len=%d", cid, payload != "", len(payload))
				} else {
					debugKitty("cid lookup skipped inline map nil for %s", cid)
				}
			} else if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
				payload = fetchRemoteBase64(src)
			}

			if payload != "" {
				if rendered := kittyInlineImage(payload); rendered != "" {
					debugKitty("rendered inline image src=%s len=%d dataURI=%t cid=%t", src, len(payload), strings.HasPrefix(src, "data:"), strings.HasPrefix(src, "cid:"))
					s.ReplaceWithHtml("\n" + rendered + "\n")
					return
				}
				debugKitty("payload present but renderer returned empty src=%s len=%d", src, len(payload))
			} else {
				debugKitty("no payload for src=%s dataURI=%t cid=%t", src, strings.HasPrefix(src, "data:"), strings.HasPrefix(src, "cid:"))
			}
		} else {
			debugKitty("kitty not detected for src=%s", src)
		}

		s.ReplaceWithHtml(hyperlink(src, fmt.Sprintf("\n [Click here to view image: %s] \n", alt)))
	})

	text := doc.Text()

	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	text = strings.TrimSpace(text)

	return bodyStyle.Render(text), nil
}
