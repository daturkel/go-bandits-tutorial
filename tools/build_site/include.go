package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// A marker looks like
//
//	<!-- include: ch02/bandit/policy.go#Policy,NewPolicy title="policy.go" hl=3-5 diff -->
//
// The fragment after # selects declarations (Name or Type.Method) or a
// `region: Name` ... `endregion: Name` block. lines=a-b selects raw lines.
var markerRE = regexp.MustCompile(`<!--\s*(include|copy|transcript|svg):\s*(.*?)\s*-->`)

type builder struct {
	root     string // repository root
	chapters []Chapter
	shown    map[string]map[int]bool // file (relative to solutions/) -> lines displayed on the current page
	names    map[string]bool         // identifiers the course has shown so far, across pages
}

func (b *builder) markShown(file string, from, to int) {
	if b.shown == nil {
		b.shown = map[string]map[int]bool{}
	}
	if b.shown[file] == nil {
		b.shown[file] = map[int]bool{}
	}
	for n := from; n <= to; n++ {
		b.shown[file][n] = true
	}
}

type codeLine struct {
	n     int
	html  string
	added bool
}

func (b *builder) expand(page string, chapterID string) (string, error) {
	var firstErr error
	out := markerRE.ReplaceAllStringFunc(page, func(m string) string {
		sub := markerRE.FindStringSubmatch(m)
		var res string
		var err error
		switch sub[1] {
		case "include":
			res, err = b.renderInclude(sub[2])
		case "copy":
			res, err = b.renderCopy(sub[2])
		case "transcript":
			res, err = b.renderTranscript(sub[2])
		case "svg":
			res, err = b.renderSVG(sub[2])
		}
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", strings.TrimSpace(m), err)
			}
			return ""
		}
		return res
	})
	return out, firstErr
}

// parseArgs splits `target key=value "quoted value" flag` into a target and options.
func parseArgs(s string) (target string, opts map[string]string, err error) {
	opts = map[string]string{}
	var fields []string
	var cur strings.Builder
	inQ := false
	for _, r := range s {
		switch {
		case r == '"':
			inQ = !inQ
			cur.WriteRune(r)
		case r == ' ' && !inQ:
			if cur.Len() > 0 {
				fields = append(fields, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		fields = append(fields, cur.String())
	}
	if len(fields) == 0 {
		return "", nil, fmt.Errorf("empty marker")
	}
	target = fields[0]
	for _, f := range fields[1:] {
		k, v, _ := strings.Cut(f, "=")
		opts[k] = strings.Trim(v, `"`)
	}
	return target, opts, nil
}

func (b *builder) renderInclude(spec string) (string, error) {
	target, opts, err := parseArgs(spec)
	if err != nil {
		return "", err
	}
	file, frag, _ := strings.Cut(target, "#")
	// Paths under exercises/ are relative to the repository root; everything
	// else is relative to solutions/.
	isExercise := strings.HasPrefix(file, "exercises/")
	base := filepath.Join(b.root, "solutions")
	if isExercise {
		base = b.root
	}
	full := filepath.Join(base, filepath.FromSlash(file))
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	src := string(data)
	lines := strings.Split(strings.TrimSuffix(src, "\n"), "\n")

	// Which lines are new compared with the previous chapter's copy of this file?
	added := make([]bool, len(lines))
	newFile := false
	if _, ok := opts["diff"]; ok && !isExercise {
		chDir, rel, _ := strings.Cut(file, "/")
		if prev := previousChapter(chDir); prev != "" {
			prevData, perr := os.ReadFile(filepath.Join(b.root, "solutions", prev, filepath.FromSlash(rel)))
			if perr == nil {
				added = addedLines(strings.Split(strings.TrimSuffix(string(prevData), "\n"), "\n"), lines)
			} else {
				newFile = true
			}
		}
	}

	// Pick the line ranges to show.
	type rng struct{ from, to int } // 1-based inclusive
	var ranges []rng
	switch {
	case opts["lines"] != "":
		from, to, err := parseRange(opts["lines"])
		if err != nil {
			return "", err
		}
		ranges = append(ranges, rng{from, min(to, len(lines))})
	case frag == "":
		ranges = append(ranges, rng{1, len(lines)})
	default:
		for _, name := range strings.Split(frag, ",") {
			from, to, err := findFragment(file, src, lines, name)
			if err != nil {
				return "", err
			}
			ranges = append(ranges, rng{from, to})
		}
	}

	hl := map[int]bool{} // 1-based snippet line numbers to highlight (explicit hl=)
	if v := opts["hl"]; v != "" {
		for _, part := range strings.Split(v, ",") {
			from, to, err := parseRange(part)
			if err != nil {
				return "", err
			}
			for i := from; i <= to; i++ {
				hl[i] = true
			}
		}
	}

	var highlighted []string
	if strings.HasSuffix(file, ".go") {
		highlighted = splitLines(highlightGo(src))
	} else {
		highlighted = splitLines(highlightPlain(src))
	}

	var shown []codeLine
	if !isExercise {
		for _, r := range ranges {
			b.markShown(file, r.from, r.to)
		}
	}
	for ri, r := range ranges {
		if ri > 0 {
			shown = append(shown, codeLine{n: 0, html: ""})
		}
		for n := r.from; n <= r.to; n++ {
			shown = append(shown, codeLine{n: n, html: highlighted[n-1], added: added[n-1]})
		}
	}
	shown = trimIndent(shown, lines)

	name := opts["title"]
	if name == "" {
		_, rel, _ := strings.Cut(file, "/")
		name = rel
		if isExercise { // exercises/chNN/exM/... -> exM/...
			parts := strings.SplitN(file, "/", 3)
			name = parts[2]
		}
	}
	meta := ""
	switch {
	case newFile:
		meta = "new file"
	case frag != "" || opts["lines"] != "":
		meta = "excerpt"
	}
	if _, ok := opts["diff"]; ok && !newFile {
		anyAdded := false
		for _, l := range shown {
			anyAdded = anyAdded || l.added
		}
		if anyAdded {
			if meta != "" {
				meta += " · "
			}
			meta += "highlighted lines are new in this chapter"
		}
	}

	var sb strings.Builder
	sb.WriteString(`<figure class="code">` + "\n")
	sb.WriteString(`<figcaption><span class="fname">` + html.EscapeString(name) + `</span>`)
	if meta != "" {
		sb.WriteString(`<span class="fmeta">` + html.EscapeString(meta) + `</span>`)
	}
	srcHref := "../solutions/" + file
	if isExercise {
		srcHref = "../" + file
	}
	sb.WriteString(`<a class="fsrc" href="` + html.EscapeString(srcHref) + `">full file</a>`)
	sb.WriteString(`<button type="button" class="copy" aria-label="Copy code">Copy</button></figcaption>` + "\n")
	sb.WriteString(`<pre tabindex="0"><code>`)
	for i, l := range shown {
		cls := "ln"
		if l.added || hl[i+1] {
			cls += " add"
		}
		if l.n == 0 {
			sb.WriteString(`<span class="ln gap" data-n="⋮"></span>`)
			continue
		}
		fmt.Fprintf(&sb, `<span class="%s" data-n="%d">%s</span>`, cls, l.n, l.html)
	}
	sb.WriteString("</code></pre>\n</figure>")
	return sb.String(), nil
}

// trimIndent is a hook for future dedenting of excerpts; lines are kept as-is
// so the shown text is byte-for-byte the source.
func trimIndent(shown []codeLine, _ []string) []codeLine { return shown }

func parseRange(s string) (from, to int, err error) {
	a, bStr, ok := strings.Cut(s, "-")
	if from, err = strconv.Atoi(a); err != nil {
		return 0, 0, fmt.Errorf("bad range %q", s)
	}
	if !ok {
		return from, from, nil
	}
	if to, err = strconv.Atoi(bStr); err != nil {
		return 0, 0, fmt.Errorf("bad range %q", s)
	}
	return from, to, nil
}

var chRE = regexp.MustCompile(`^ch(\d+)$`)

func previousChapter(id string) string {
	m := chRE.FindStringSubmatch(id)
	if m == nil {
		return ""
	}
	n, _ := strconv.Atoi(m[1])
	if n <= 1 {
		return ""
	}
	return fmt.Sprintf("ch%02d", n-1)
}

// findFragment locates a region marker or a Go declaration and returns its
// 1-based inclusive line range.
func findFragment(file, src string, lines []string, name string) (int, int, error) {
	start := -1
	for i, l := range lines {
		if strings.Contains(l, "region: "+name) && !strings.Contains(l, "endregion") {
			start = i + 1
		}
		if start > 0 && strings.Contains(l, "endregion: "+name) {
			return start + 1, i, nil
		}
	}
	if !strings.HasSuffix(file, ".go") {
		return 0, 0, fmt.Errorf("no region %q in %s", name, file)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return 0, 0, err
	}
	recv, member, isMethod := strings.Cut(name, ".")
	for _, d := range f.Decls {
		var doc *ast.CommentGroup
		match := false
		switch d := d.(type) {
		case *ast.FuncDecl:
			doc = d.Doc
			if isMethod {
				match = d.Recv != nil && d.Name.Name == member && recvName(d.Recv) == recv
			} else {
				match = d.Recv == nil && d.Name.Name == name
			}
		case *ast.GenDecl:
			doc = d.Doc
			if !isMethod {
				for _, sp := range d.Specs {
					switch sp := sp.(type) {
					case *ast.TypeSpec:
						match = match || sp.Name.Name == name
					case *ast.ValueSpec:
						for _, n := range sp.Names {
							match = match || n.Name == name
						}
					}
				}
			}
		}
		if match {
			from := d.Pos()
			if doc != nil {
				from = doc.Pos()
			}
			return fset.Position(from).Line, fset.Position(d.End()).Line, nil
		}
	}
	return 0, 0, fmt.Errorf("no declaration or region %q in %s", name, file)
}

func recvName(fl *ast.FieldList) string {
	if len(fl.List) == 0 {
		return ""
	}
	t := fl.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	if ix, ok := t.(*ast.IndexExpr); ok {
		t = ix.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// addedLines marks lines of cur that are not part of a longest common
// subsequence with prev.
func addedLines(prev, cur []string) []bool {
	n, m := len(prev), len(cur)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if prev[i] == cur[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	added := make([]bool, m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case prev[i] == cur[j]:
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			i++
		default:
			added[j] = true
			j++
		}
	}
	for ; j < m; j++ {
		added[j] = true
	}
	// Blank lines and lone braces are noise when they are the only change.
	for k, l := range cur {
		if added[k] && strings.TrimSpace(l) == "" {
			added[k] = false
		}
	}
	return added
}

func (b *builder) renderTranscript(spec string) (string, error) {
	target, _, err := parseArgs(spec)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(b.root, "site", "generated", filepath.FromSlash(target)+".txt"))
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(`<figure class="term"><figcaption><span class="fname">terminal</span></figcaption>` + "\n<pre tabindex=\"0\"><code>")
	for _, l := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if cmd, ok := strings.CutPrefix(l, "$ "); ok {
			sb.WriteString(`<span class="ln cmd">` + html.EscapeString(cmd) + `</span>`)
		} else {
			sb.WriteString(`<span class="ln out">` + html.EscapeString(l) + `</span>`)
		}
	}
	sb.WriteString("</code></pre></figure>")
	return sb.String(), nil
}

func (b *builder) renderSVG(spec string) (string, error) {
	target, opts, err := parseArgs(spec)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(b.root, "site", "generated", filepath.FromSlash(target)))
	if err != nil {
		return "", err
	}
	svg := strings.Replace(string(data), "<svg ", "<svg data-embedded ", 1)
	// Unique ids so two inline charts never collide.
	id := strings.NewReplacer("/", "-", ".", "-").Replace(target)
	svg = strings.ReplaceAll(svg, "rc-title", id+"-title")
	svg = strings.ReplaceAll(svg, "rc-desc", id+"-desc")
	caption := opts["caption"]
	var sb strings.Builder
	sb.WriteString(`<figure class="chart">` + svg)
	src := "generated/" + path.Clean(target)
	sb.WriteString(`<figcaption>` + html.EscapeString(caption))
	sb.WriteString(` <a href="` + src + `">Standalone SVG</a></figcaption></figure>`)
	return sb.String(), nil
}

// renderCopy handles a file the reader is told to copy rather than type. It
// counts as covering every line of the file for the coverage check.
func (b *builder) renderCopy(spec string) (string, error) {
	target, opts, err := parseArgs(spec)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(b.root, "solutions", filepath.FromSlash(target)))
	if err != nil {
		return "", err
	}
	n := len(strings.Split(strings.TrimSuffix(string(data), "\n"), "\n"))
	b.markShown(target, 1, n)
	_, rel, _ := strings.Cut(target, "/")
	why := opts["why"]
	if why == "" {
		why = "it is long and not about Go itself"
	}
	return fmt.Sprintf(`<aside class="callout note"><p class="callout-title">Copy this file</p><p>Copy <a href="../solutions/%s"><code>%s</code></a> from <code>solutions/%s</code> instead of typing it; %s.</p></aside>`,
		html.EscapeString(target), html.EscapeString(rel), html.EscapeString(strings.SplitN(target, "/", 2)[0]), html.EscapeString(why)), nil
}

// checkCoverage fails if a chapter adds source lines that its page neither
// shows nor tells the reader to copy. Test files, _examples and import blocks
// are exempt: tests are supplied by tools/check.sh, and gopls adds imports.
func (b *builder) checkCoverage(chapterID string) error {
	dir := filepath.Join(b.root, "solutions", chapterID)
	prev := previousChapter(chapterID)
	var problems []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), "_") || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		added := make([]bool, len(lines))
		for i := range added {
			added[i] = true
		}
		if prev != "" {
			if pd, perr := os.ReadFile(filepath.Join(b.root, "solutions", prev, rel)); perr == nil {
				added = addedLines(strings.Split(strings.TrimSuffix(string(pd), "\n"), "\n"), lines)
			}
		}
		exempt := packageAndImportLines(string(data))
		key := chapterID + "/" + filepath.ToSlash(rel)
		var missing []int
		for i := range lines {
			n := i + 1
			if added[i] && !exempt[n] && !b.shown[key][n] && strings.TrimSpace(lines[i]) != "" {
				missing = append(missing, n)
			}
		}
		if len(missing) > 0 {
			problems = append(problems, fmt.Sprintf("%s: lines not shown or copied: %s", key, ranges(missing)))
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s adds source the page does not show:\n  %s", chapterID, strings.Join(problems, "\n  "))
	}
	return nil
}

func packageAndImportLines(src string) map[int]bool {
	out := map[int]bool{}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ImportsOnly)
	if err != nil {
		return out
	}
	mark := func(from, to token.Pos) {
		for n := fset.Position(from).Line; n <= fset.Position(to).Line; n++ {
			out[n] = true
		}
	}
	mark(f.Package, f.Name.End())
	for _, d := range f.Decls {
		mark(d.Pos(), d.End())
	}
	return out
}

func ranges(ns []int) string {
	var parts []string
	for i := 0; i < len(ns); {
		j := i
		for j+1 < len(ns) && ns[j+1] == ns[j]+1 {
			j++
		}
		if i == j {
			parts = append(parts, strconv.Itoa(ns[i]))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", ns[i], ns[j]))
		}
		i = j + 1
	}
	return strings.Join(parts, ", ")
}

// collectShownNames adds every identifier appearing in this page's displayed
// lines to the cumulative set of names the reader has been shown.
func (b *builder) collectShownNames() {
	if b.names == nil {
		b.names = map[string]bool{}
	}
	for file, lines := range b.shown {
		data, err := os.ReadFile(filepath.Join(b.root, "solutions", filepath.FromSlash(file)))
		if err != nil || !strings.HasSuffix(file, ".go") {
			continue
		}
		src := strings.Split(string(data), "\n")
		for n := range lines {
			if n >= 1 && n <= len(src) {
				for _, id := range identRE.FindAllString(src[n-1], -1) {
					b.names[id] = true
				}
			}
		}
	}
}

var identRE = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// checkTestNames fails if the chapter's reference tests use a name that is
// declared in the chapter's non-test code but has not been shown on this page
// or an earlier one. Those are the names a reader must spell exactly.
func (b *builder) checkTestNames(chapterID string) error {
	dir := filepath.Join(b.root, "solutions", chapterID)
	declared := map[string]bool{}
	type use struct{ name, file string }
	var uses []use
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), "_") || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		rel, _ := filepath.Rel(dir, path)
		if strings.HasSuffix(path, "_test.go") {
			ast.Inspect(f, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.Ident:
					uses = append(uses, use{n.Name, rel})
				}
				return true
			})
			return nil
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				declared[d.Name.Name] = true
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					switch sp := sp.(type) {
					case *ast.TypeSpec:
						declared[sp.Name.Name] = true
						ast.Inspect(sp.Type, func(n ast.Node) bool {
							if fl, ok := n.(*ast.Field); ok {
								for _, nm := range fl.Names {
									declared[nm.Name] = true
								}
							}
							return true
						})
					case *ast.ValueSpec:
						for _, nm := range sp.Names {
							declared[nm.Name] = true
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	missing := map[string]string{}
	for _, u := range uses {
		if declared[u.name] && !b.names[u.name] && u.name != "main" && u.name != "_" {
			if _, ok := missing[u.name]; !ok {
				missing[u.name] = u.file
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	var parts []string
	for name, file := range missing {
		parts = append(parts, fmt.Sprintf("%s (used in %s)", name, file))
	}
	sort.Strings(parts)
	return fmt.Errorf("%s: reference tests use names no page has shown:\n  %s", chapterID, strings.Join(parts, "\n  "))
}

// renderTests lists the reference test files that tools/check.sh applies.
func (b *builder) renderTests(chapterID string) string {
	dir := filepath.Join(b.root, "solutions", chapterID)
	var items []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(rel, "_test.go") || strings.Contains(filepath.ToSlash(rel), "testdata/") {
			items = append(items, fmt.Sprintf(`<li><a href="../solutions/%s/%s"><code>%s</code></a></li>`, chapterID, filepath.ToSlash(rel), filepath.ToSlash(rel)))
		}
		return nil
	})
	if len(items) == 0 {
		return ""
	}
	return `<details class="tests-list"><summary>Reference tests <code>check.sh</code> runs (you do not need to write these)</summary><ul>` + strings.Join(items, "") + `</ul></details>`
}
