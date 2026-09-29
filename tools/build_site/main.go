// build_site turns site/src/*.html page sources into finished pages in site/.
//
//	go run ./tools/build_site        (from the repository root, via `cd tools/build_site && go run . ../..`)
//
// It expands include/transcript/svg markers from real files, highlights code,
// wraps pages in the shared layout, and finally checks every internal link.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Chapter is one entry of site/src/chapters.json.
type Chapter struct {
	ID      string `json:"id"`    // ch01, ..., capstone
	Label   string `json:"label"` // "Chapter 1" or "Capstone"
	Title   string `json:"title"`
	Part    string `json:"part"`
	Summary string `json:"summary"`
	Status  string `json:"status"` // ready | planned
}

func (c Chapter) Ready() bool  { return c.Status == "ready" }
func (c Chapter) File() string { return c.ID + ".html" }

// Number is the short reel label shown next to the title: "01" or "★".
func (c Chapter) Number() string {
	if strings.HasPrefix(c.ID, "ch") {
		return c.ID[2:]
	}
	return "★"
}

type tocEntry struct{ ID, Text string }

type pageData struct {
	Title       string
	Description string
	Chapter     *Chapter
	Prev, Next  *Chapter
	Chapters    []Chapter
	Body        template.HTML
	TOC         []tocEntry
	IsIndex     bool
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: build_site <repo-root>")
		os.Exit(2)
	}
	if err := build(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "build_site:", err)
		os.Exit(1)
	}
}

func build(root string) error {
	raw, err := os.ReadFile(filepath.Join(root, "site", "src", "chapters.json"))
	if err != nil {
		return err
	}
	var chapters []Chapter
	if err := json.Unmarshal(raw, &chapters); err != nil {
		return fmt.Errorf("chapters.json: %w", err)
	}
	layout, err := template.ParseFiles(filepath.Join(root, "site", "src", "_layout.html"))
	if err != nil {
		return err
	}
	b := &builder{root: root, chapters: chapters}
	var coverageErrs []error

	render := func(name string, pd pageData, outName string) error {
		src, err := os.ReadFile(filepath.Join(root, "site", "src", name))
		if err != nil {
			return err
		}
		b.shown = nil
		body, err := b.expand(string(src), pd.Title)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if pd.Chapter != nil && strings.HasPrefix(pd.Chapter.ID, "ch") {
			if err := b.checkCoverage(pd.Chapter.ID); err != nil {
				coverageErrs = append(coverageErrs, err)
			}
			b.collectShownNames()
			if err := b.checkTestNames(pd.Chapter.ID); err != nil {
				coverageErrs = append(coverageErrs, err)
			}
			body = strings.Replace(body, "<!-- tests -->", b.renderTests(pd.Chapter.ID), 1)
		}
		if pd.IsIndex {
			body = strings.Replace(body, "<!-- chapters -->", chapterList(chapters), 1)
			start := "Chapter 1 (not published yet)"
			for _, c := range chapters {
				if c.Ready() {
					start = fmt.Sprintf(`<a href="%s">%s</a>`, c.File(), c.Label)
					break
				}
			}
			body = strings.Replace(body, "<!-- first-chapter -->", start, 1)
		}
		body, toc := addHeadingIDs(body)
		pd.Body, pd.TOC, pd.Chapters = template.HTML(body), toc, chapters
		var buf bytes.Buffer
		if err := layout.Execute(&buf, pd); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, "site", outName), buf.Bytes(), 0o644)
	}

	if err := render("index.html", pageData{Title: "Go Bandits", Description: "Learn Go by building a multi-armed bandit service, from a CLI simulation to replicated microservices.", IsIndex: true}, "index.html"); err != nil {
		return err
	}
	for i := range chapters {
		c := &chapters[i]
		if !c.Ready() {
			// Drop any stale page from when the chapter was ready.
			if err := os.Remove(filepath.Join(root, "site", c.File())); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		pd := pageData{Title: c.Title, Description: c.Summary, Chapter: c}
		if i > 0 {
			pd.Prev = &chapters[i-1]
		}
		if i+1 < len(chapters) {
			pd.Next = &chapters[i+1]
		}
		if err := render(c.ID+".html", pd, c.File()); err != nil {
			return err
		}
	}
	if err := errors.Join(coverageErrs...); err != nil {
		return err
	}
	return checkLinks(root)
}

func chapterList(chapters []Chapter) string {
	var sb strings.Builder
	part := ""
	open := false
	for _, c := range chapters {
		if c.Part != part {
			if open {
				sb.WriteString("</ol>\n</section>\n")
			}
			part = c.Part
			fmt.Fprintf(&sb, "<section class=\"part\">\n<h2>%s</h2>\n<ol class=\"chapter-list\">\n", html.EscapeString(part))
			open = true
		}
		cls := "chapter-card"
		tag := "planned"
		if c.Ready() {
			tag = "ready"
		}
		fmt.Fprintf(&sb, "<li class=\"%s %s\">", cls, tag)
		inner := fmt.Sprintf(`<span class="reel" aria-hidden="true">%s</span><span class="card-body"><span class="card-title">%s</span><span class="card-sum">%s</span></span><span class="badge">%s</span>`,
			c.Number(), html.EscapeString(c.Title), html.EscapeString(c.Summary), map[bool]string{true: "Ready", false: "Planned"}[c.Ready()])
		if c.Ready() {
			fmt.Fprintf(&sb, `<a href="%s">%s</a>`, c.File(), inner)
		} else {
			fmt.Fprintf(&sb, `<div class="planned-card">%s</div>`, inner)
		}
		sb.WriteString("</li>\n")
	}
	if open {
		sb.WriteString("</ol>\n</section>\n")
	}
	return sb.String()
}

var h2RE = regexp.MustCompile(`(?s)<h2(\s[^>]*)?>(.*?)</h2>`)
var tagRE = regexp.MustCompile(`<[^>]+>`)
var idAttrRE = regexp.MustCompile(`\bid="([^"]+)"`)

func slugify(s string) string {
	s = strings.ToLower(html.UnescapeString(tagRE.ReplaceAllString(s, "")))
	var sb strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			dash = false
		} else if !dash && sb.Len() > 0 {
			sb.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(sb.String(), "-")
}

// addHeadingIDs gives every <h2> an id and returns the table of contents.
func addHeadingIDs(body string) (string, []tocEntry) {
	var toc []tocEntry
	used := map[string]bool{}
	out := h2RE.ReplaceAllStringFunc(body, func(m string) string {
		sub := h2RE.FindStringSubmatch(m)
		attrs, inner := sub[1], sub[2]
		id := ""
		if am := idAttrRE.FindStringSubmatch(attrs); am != nil {
			id = am[1]
		} else {
			id = slugify(inner)
			for base, n := id, 2; used[id]; n++ {
				id = fmt.Sprintf("%s-%d", base, n)
			}
			attrs += ` id="` + id + `"`
		}
		used[id] = true
		toc = append(toc, tocEntry{ID: id, Text: html.UnescapeString(tagRE.ReplaceAllString(inner, ""))})
		return "<h2" + attrs + ">" + inner + "</h2>"
	})
	return out, toc
}
