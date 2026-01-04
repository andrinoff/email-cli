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
	"golang.org/x/sys/unix"
)

// getTerminalCellSize returns the height of a terminal cell in pixels.
// It queries the terminal using TIOCGWINSZ to get both character and pixel dimensions.
// Falls back to a default of 18 pixels if the query fails.
func getTerminalCellSize() int {
	const defaultCellHeight = 18

	// Try stdout, stdin, stderr, then /dev/tty as last resort
	fds := []int{int(os.Stdout.Fd()), int(os.Stdin.Fd()), int(os.Stderr.Fd())}

	for _, fd := range fds {
		if cellHeight := getCellHeightFromFd(fd); cellHeight > 0 {
			return cellHeight
		}
	}

	// Try /dev/tty directly - this works even when stdio is redirected (e.g., in Bubble Tea)
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		if cellHeight := getCellHeightFromFd(int(tty.Fd())); cellHeight > 0 {
			return cellHeight
		}
	}

	debugImageProtocol("using default cell height: %d pixels", defaultCellHeight)
	return defaultCellHeight
}

// getCellHeightFromFd attempts to get the terminal cell height from a file descriptor.
// Returns 0 if it fails or if pixel dimensions are not available.
func getCellHeightFromFd(fd int) int {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return 0
	}

	// ws.Row = number of character rows
	// ws.Ypixel = height in pixels
	// Some terminals don't report pixel dimensions (return 0)
	if ws.Row > 0 && ws.Ypixel > 0 {
		cellHeight := int(ws.Ypixel) / int(ws.Row)
		if cellHeight > 0 {
			debugImageProtocol("terminal cell height: %d pixels (rows=%d, ypixel=%d, fd=%d)", cellHeight, ws.Row, ws.Ypixel, fd)
			return cellHeight
		}
	}

	// Terminal reported dimensions but no pixel info - this is common
	if ws.Row > 0 && ws.Ypixel == 0 {
		debugImageProtocol("terminal fd=%d has rows=%d but no pixel info (ypixel=0)", fd, ws.Row)
	}

	return 0
}

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

func ghosttySupported() bool {
	// Check for TERM containing ghostty
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "ghostty") {
		return true
	}

	// Check for Ghostty-specific environment variables
	if os.Getenv("TERM_PROGRAM") == "ghostty" {
		return true
	}

	// Check for GHOSTTY_RESOURCES_DIR which Ghostty sets
	return os.Getenv("GHOSTTY_RESOURCES_DIR") != ""
}

// imageProtocolSupported checks if any supported image protocol terminal is detected.
func imageProtocolSupported() bool {
	return kittySupported() || ghosttySupported()
}

func debugImageProtocol(format string, args ...interface{}) {
	if os.Getenv("DEBUG_IMAGE_PROTOCOL") == "" && os.Getenv("DEBUG_KITTY_IMAGES") == "" {
		return
	}
	msg := fmt.Sprintf("[img-protocol] "+format+"\n", args...)
	fmt.Print(msg)
	if path := os.Getenv("DEBUG_IMAGE_PROTOCOL_LOG"); path != "" {
		if f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(msg)
			_ = f.Close()
		}
	} else if path := os.Getenv("DEBUG_KITTY_LOG"); path != "" {
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
		debugImageProtocol("remote fetch failed url=%s err=%v", url, err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		debugImageProtocol("remote fetch non-200 url=%s status=%d", url, resp.StatusCode)
		return ""
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		debugImageProtocol("remote fetch read error url=%s err=%v", url, err)
		return ""
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		debugImageProtocol("remote decode failed url=%s err=%v", url, err)
		return ""
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		debugImageProtocol("remote png encode failed url=%s err=%v", url, err)
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	debugImageProtocol("remote fetch ok url=%s len=%d", url, len(encoded))
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

// imageRowPlaceholderPrefix is used to mark where image row spacing should be inserted.
// This prevents the newline-collapsing regex from removing intentional spacing.
// Uses brackets instead of angle brackets to avoid being interpreted as HTML tags.
const imageRowPlaceholderPrefix = "[[MATCHA_IMG_ROWS:"
const imageRowPlaceholderSuffix = "]]"

func kittyInlineImage(payload string) string {
	if payload == "" {
		return ""
	}

	const chunkSize = 4096
	var b strings.Builder

	// Calculate how many terminal rows the image occupies to advance text after it.
	rows := 1
	if data, err := base64.StdEncoding.DecodeString(payload); err == nil {
		if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
			cellHeight := getTerminalCellSize()
			h := img.Bounds().Dy()
			rows = (h + cellHeight - 1) / cellHeight
			if rows < 1 {
				rows = 1
			}
			debugImageProtocol("image height: %d pixels, cell height: %d pixels, rows needed: %d", h, cellHeight, rows)
		}
	}

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
			// C=1 means cursor does NOT move after image render (stays at top-left of image position)
			// This is needed for proper TUI rendering, but we must add newlines to push text below
			b.WriteString(fmt.Sprintf("\x1b_Gf=100,a=T,q=2,C=1,m=%s;%s\x1b\\", more, chunk))
		} else {
			b.WriteString(fmt.Sprintf("\x1b_Gm=%s;%s\x1b\\", more, chunk))
		}
	}

	// Add newlines to push cursor below the image.
	// Use a placeholder that won't be collapsed by the newline regex.
	b.WriteString(fmt.Sprintf("\n%s%d%s\n", imageRowPlaceholderPrefix, rows, imageRowPlaceholderSuffix))

	return b.String()
}

// expandImageRowPlaceholders replaces image row placeholders with actual newlines.
func expandImageRowPlaceholders(text string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(imageRowPlaceholderPrefix) + `(\d+)` + regexp.QuoteMeta(imageRowPlaceholderSuffix))
	return re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the number of rows from the placeholder
		numStr := strings.TrimPrefix(match, imageRowPlaceholderPrefix)
		numStr = strings.TrimSuffix(numStr, imageRowPlaceholderSuffix)
		rows := 1
		if _, err := fmt.Sscanf(numStr, "%d", &rows); err != nil || rows < 1 {
			rows = 1
		}
		// Return the newlines needed to push content below the image
		return strings.Repeat("\n", rows)
	})
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

		if imageProtocolSupported() {
			var payload string
			if strings.HasPrefix(src, "data:image/") {
				payload = dataURIBase64(src)
			} else if strings.HasPrefix(src, "cid:") {
				cid := strings.TrimPrefix(src, "cid:")
				cid = strings.Trim(cid, "<>")
				if inline != nil {
					payload = inline[cid]
					debugImageProtocol("cid lookup for %s found=%t len=%d", cid, payload != "", len(payload))
				} else {
					debugImageProtocol("cid lookup skipped inline map nil for %s", cid)
				}
			} else if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
				payload = fetchRemoteBase64(src)
			}

			if payload != "" {
				if rendered := kittyInlineImage(payload); rendered != "" {
					debugImageProtocol("rendered inline image src=%s len=%d dataURI=%t cid=%t (kitty=%t ghostty=%t)", src, len(payload), strings.HasPrefix(src, "data:"), strings.HasPrefix(src, "cid:"), kittySupported(), ghosttySupported())
					s.ReplaceWithHtml("\n" + rendered + "\n")
					return
				}
				debugImageProtocol("payload present but renderer returned empty src=%s len=%d", src, len(payload))
			} else {
				debugImageProtocol("no payload for src=%s dataURI=%t cid=%t", src, strings.HasPrefix(src, "data:"), strings.HasPrefix(src, "cid:"))
			}
		} else {
			debugImageProtocol("image protocol not supported for src=%s (kitty=%t ghostty=%t)", src, kittySupported(), ghosttySupported())
		}

		s.ReplaceWithHtml(hyperlink(src, fmt.Sprintf("\n [Click here to view image: %s] \n", alt)))
	})

	text := doc.Text()

	// Collapse excessive newlines, but not the image row placeholders
	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	// Now expand the image row placeholders to actual newlines
	text = expandImageRowPlaceholders(text)

	return bodyStyle.Render(text), nil
}
