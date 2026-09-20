package mailtpl

import (
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const DefaultAccent = "#6366f1"

var (
	hexColor    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	htmlTag     = regexp.MustCompile(`(?s)<[^>]*>`)
	blankLines  = regexp.MustCompile("\n{3,}")
	lineBreaks  = regexp.MustCompile(`(?i)<br\s*/?>`)
	blockBreaks = regexp.MustCompile(`(?i)</(p|div|tr|h[1-6]|li)>`)
)

type Brand struct {
	Name    string
	LogoURL string
	Accent  string
}

type Message struct {
	Title       string
	Body        string
	ActionLabel string
	ActionURL   string
}

func (b Brand) accent() string {
	if v := strings.TrimSpace(b.Accent); hexColor.MatchString(v) {
		return strings.ToLower(v)
	}
	return DefaultAccent
}

func (b Brand) name() string {
	if v := strings.TrimSpace(b.Name); v != "" {
		return v
	}
	return "Vortanix"
}

func readableText(hex string) string {
	var channels [3]float64
	for i := range channels {
		v, _ := strconv.ParseUint(hex[1+i*2:3+i*2], 16, 8)
		c := float64(v) / 255
		if c <= 0.03928 {
			channels[i] = c / 12.92
		} else {
			channels[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	luminance := 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
	if (luminance+0.05)/0.0533 > 1.05/(luminance+0.05) {
		return "#0a0b0d"
	}
	return "#ffffff"
}

func Paragraphs(text string) string {
	var b strings.Builder
	for _, p := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		b.WriteString(`<p style="margin:0 0 14px">`)
		b.WriteString(strings.ReplaceAll(html.EscapeString(p), "\n", "<br>"))
		b.WriteString(`</p>`)
	}
	return b.String()
}

func Render(b Brand, m Message) string {
	accent := b.accent()
	name := html.EscapeString(b.name())

	header := `<div style="font-size:20px;font-weight:700;letter-spacing:.02em;color:` + accent + `">` + name + `</div>`
	if logo := strings.TrimSpace(b.LogoURL); strings.HasPrefix(logo, "https://") || strings.HasPrefix(logo, "http://") {
		header = `<img src="` + html.EscapeString(logo) + `" alt="` + name + `" style="display:block;max-height:40px;max-width:220px;border:0">`
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>`)
	sb.WriteString(`<body style="margin:0;padding:0;background:#f4f5f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;color:#16171a">`)
	sb.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f4f5f7;padding:32px 12px"><tr><td align="center">`)
	sb.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:560px;background:#ffffff;border-radius:12px;border-top:4px solid ` + accent + `">`)
	sb.WriteString(`<tr><td style="padding:28px 32px 8px">` + header + `</td></tr>`)
	sb.WriteString(`<tr><td style="padding:12px 32px 28px;font-size:15px;line-height:1.6">`)
	if title := strings.TrimSpace(m.Title); title != "" {
		sb.WriteString(`<h1 style="margin:0 0 16px;font-size:20px;line-height:1.3">` + html.EscapeString(title) + `</h1>`)
	}
	sb.WriteString(m.Body)
	if url := strings.TrimSpace(m.ActionURL); url != "" && strings.TrimSpace(m.ActionLabel) != "" {
		sb.WriteString(`<p style="margin:24px 0 8px"><a href="` + html.EscapeString(url) + `" style="display:inline-block;padding:12px 22px;border-radius:8px;background:` + accent + `;color:` + readableText(accent) + `;font-weight:600;text-decoration:none">` + html.EscapeString(m.ActionLabel) + `</a></p>`)
		sb.WriteString(`<p style="margin:12px 0 0;font-size:12px;color:#8a8c90;word-break:break-all">` + html.EscapeString(url) + `</p>`)
	}
	sb.WriteString(`</td></tr></table>`)
	sb.WriteString(`<p style="margin:16px 0 0;font-size:12px;color:#8a8c90">` + name + `</p>`)
	sb.WriteString(`</td></tr></table></body></html>`)
	return sb.String()
}

func PlainText(b Brand, m Message) string {
	parts := make([]string, 0, 4)
	if title := strings.TrimSpace(m.Title); title != "" {
		parts = append(parts, title)
	}
	if body := StripHTML(m.Body); body != "" {
		parts = append(parts, body)
	}
	if url := strings.TrimSpace(m.ActionURL); url != "" {
		if label := strings.TrimSpace(m.ActionLabel); label != "" {
			parts = append(parts, label+": "+url)
		} else {
			parts = append(parts, url)
		}
	}
	parts = append(parts, b.name())
	return strings.Join(parts, "\n\n")
}

func StripHTML(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = lineBreaks.ReplaceAllString(s, "\n")
	s = blockBreaks.ReplaceAllString(s, "\n\n")
	s = htmlTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(blankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}
