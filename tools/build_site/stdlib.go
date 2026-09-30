package main

import (
	"fmt"
	"go/ast"
	gobuild "go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Standard library documentation links.
//
// A reader who has to call a standard library function in a task should be
// told where it is documented the first time it comes up. The check below
// enforces that: for every exercise block on a page of the primer or of a
// chapter in the start/solution layout, it finds the standard library symbols
// (functions, methods, types, package variables and constants) that the
// block's reference answer uses, and requires a pkg.go.dev link to each one
// that no earlier exercise used, inside that block. It also verifies that
// every pkg.go.dev link to a standard package names a symbol that exists.

var (
	exerciseStartRE = regexp.MustCompile(`<details class="exercise"`)
	docLinkRE       = regexp.MustCompile(`https://pkg\.go\.dev/([^"#\s]+)(?:#([A-Za-z0-9_.]+))?`)
)

// stdChecker holds the state shared by all pages: the symbols earlier
// exercises have already used, and the loaded packages.
type stdChecker struct {
	b    *builder
	fset *token.FileSet
	std  types.Importer
	seen map[string]bool // "path#Name" of symbols used by earlier exercises
	pkgs map[string]*loadedPkg
}

type loadedPkg struct {
	files map[string]*ast.File // by absolute file name
	info  *types.Info
	pkg   *types.Package
}

func newStdChecker(b *builder) *stdChecker {
	fset := token.NewFileSet()
	return &stdChecker{
		b:    b,
		fset: fset,
		std:  importer.ForCompiler(fset, "source", nil),
		seen: map[string]bool{},
		pkgs: map[string]*loadedPkg{},
	}
}

func isStdPath(p string) bool {
	first, _, _ := strings.Cut(p, "/")
	if strings.Contains(first, ".") {
		return false
	}
	st, err := os.Stat(filepath.Join(gobuild.Default.GOROOT, "src", filepath.FromSlash(p)))
	return err == nil && st.IsDir()
}

// localImporter resolves standard library imports with the source importer and
// imports of the module being checked from its directory.
type localImporter struct {
	c       *stdChecker
	modPath string
	modDir  string
}

func (li localImporter) Import(path string) (*types.Package, error) {
	if li.modPath != "" && (path == li.modPath || strings.HasPrefix(path, li.modPath+"/")) {
		dir := filepath.Join(li.modDir, filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(path, li.modPath), "/")))
		lp, err := li.c.load(dir, false)
		if err != nil {
			return nil, err
		}
		return lp.pkg, nil
	}
	return li.c.std.Import(path)
}

func moduleOf(dir string) (modPath, modDir string) {
	for d := dir; ; d = filepath.Dir(d) {
		if data, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			for _, l := range strings.Split(string(data), "\n") {
				if rest, ok := strings.CutPrefix(strings.TrimSpace(l), "module "); ok {
					return strings.TrimSpace(rest), d
				}
			}
			return "", d
		}
		if d == filepath.Dir(d) {
			return "", ""
		}
	}
}

// load parses and type-checks the package in dir. With tests, _test.go files
// of the package itself are included.
func (c *stdChecker) load(dir string, tests bool) (*loadedPkg, error) {
	key := fmt.Sprintf("%s|%v", dir, tests)
	if lp, ok := c.pkgs[key]; ok {
		return lp, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	lp := &loadedPkg{files: map[string]*ast.File{}}
	var files []*ast.File
	pkgName := ""
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || (!tests && strings.HasSuffix(name, "_test.go")) {
			continue
		}
		abs := filepath.Join(dir, name)
		f, err := parser.ParseFile(c.fset, abs, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(f.Name.Name, "_test") {
			continue // external test package
		}
		if pkgName == "" {
			pkgName = f.Name.Name
		}
		lp.files[abs] = f
		files = append(files, f)
	}
	modPath, modDir := moduleOf(dir)
	lp.info = &types.Info{
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	conf := types.Config{Importer: localImporter{c, modPath, modDir}, Error: func(error) {}}
	lp.pkg, _ = conf.Check(pkgName, c.fset, files, lp.info)
	if lp.pkg == nil {
		return nil, fmt.Errorf("cannot type-check %s", dir)
	}
	c.pkgs[key] = lp
	return lp, nil
}

// symbolKey names a standard library object: "import/path#Name" or
// "import/path#Type.Method". It returns "" for anything else.
func symbolKey(obj types.Object) string {
	if obj == nil || obj.Pkg() == nil || !obj.Exported() || !isStdPath(obj.Pkg().Path()) {
		return ""
	}
	pkg := obj.Pkg()
	switch o := obj.(type) {
	case *types.Func:
		if sig, ok := o.Type().(*types.Signature); ok && sig.Recv() != nil {
			return "" // methods are named through their selector; see methodKey
		}
		if o.Parent() == pkg.Scope() {
			return pkg.Path() + "#" + o.Name()
		}
	case *types.Var, *types.Const, *types.TypeName:
		if obj.Parent() == pkg.Scope() {
			return pkg.Path() + "#" + obj.Name()
		}
	}
	return ""
}

// exerciseBlocks splits a page source into its exercise blocks: each
// <details class="exercise"> element up to its matching </details>.
func exerciseBlocks(src string) []string {
	var blocks []string
	for _, s := range exerciseStartRE.FindAllStringIndex(src, -1) {
		depth, i := 0, s[0]
		for i < len(src) {
			open := strings.Index(src[i:], "<details")
			closeAt := strings.Index(src[i:], "</details>")
			if closeAt < 0 {
				i = len(src)
				break
			}
			if open >= 0 && open < closeAt {
				depth++
				i += open + len("<details")
				continue
			}
			depth--
			i += closeAt + len("</details>")
			if depth == 0 {
				break
			}
		}
		blocks = append(blocks, src[s[0]:i])
	}
	return blocks
}

func blockTitle(block string) string {
	i := strings.Index(block, "<summary>")
	j := strings.Index(block, "</summary>")
	if i < 0 || j < i {
		return "(untitled)"
	}
	return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(block[i+len("<summary>"):j], "")
}

// answerSymbols returns the standard library symbols used by the reference
// answers included in a block.
func (c *stdChecker) answerSymbols(block string) ([]string, error) {
	set := map[string]bool{}
	for _, m := range markerRE.FindAllStringSubmatch(block, -1) {
		if m[1] != "include" {
			continue
		}
		target, opts, err := parseArgs(m[2])
		if err != nil {
			return nil, err
		}
		file, frag, _ := strings.Cut(target, "#")
		if !strings.HasSuffix(file, ".go") || startRE.MatchString(file) {
			continue
		}
		var full string
		switch {
		case strings.HasPrefix(file, "primer/") || strings.HasPrefix(file, "exercises/"):
			if !strings.Contains(file, "/reference/") {
				continue
			}
			full = filepath.Join(c.b.root, filepath.FromSlash(file))
		default:
			full = c.b.solutionPath(file)
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		src := string(data)
		lines := strings.Split(strings.TrimSuffix(src, "\n"), "\n")
		type span struct{ from, to int }
		var spans []span
		switch {
		case opts["lines"] != "":
			from, to, err := parseRange(opts["lines"])
			if err != nil {
				return nil, err
			}
			spans = append(spans, span{from, to})
		case frag == "":
			spans = append(spans, span{1, len(lines)})
		default:
			for _, name := range strings.Split(frag, ",") {
				from, to, err := findFragment(file, src, lines, name)
				if err != nil {
					return nil, err
				}
				spans = append(spans, span{from, to})
			}
		}
		absFull, _ := filepath.Abs(full)
		lp, err := c.load(filepath.Dir(absFull), true)
		if err != nil {
			return nil, err
		}
		f := lp.files[absFull]
		if f == nil {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			var key string
			var pos token.Pos
			switch n := n.(type) {
			case *ast.Ident:
				key, pos = symbolKey(lp.info.Uses[n]), n.Pos()
			case *ast.SelectorExpr:
				key, pos = methodKey(lp.info.Selections[n]), n.Sel.Pos()
			}
			if key == "" {
				return true
			}
			line := c.fset.Position(pos).Line
			for _, s := range spans {
				if line >= s.from && line <= s.to {
					set[key] = true
				}
			}
			return true
		})
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// linked reports whether the block links to the documentation of the symbol
// (or, for a method, of its type).
func linked(block, key string) bool {
	path, name, _ := strings.Cut(key, "#")
	typ, _, isMethod := strings.Cut(name, ".")
	for _, m := range docLinkRE.FindAllStringSubmatch(block, -1) {
		if m[1] != path {
			continue
		}
		if m[2] == name || (isMethod && m[2] == typ) {
			return true
		}
	}
	return false
}

// checkDocLink verifies that a pkg.go.dev link names a standard package and
// an existing symbol.
func (c *stdChecker) checkDocLink(path, frag string) error {
	if path == "std" {
		return nil // the index of the standard library
	}
	if !isStdPath(path) {
		return fmt.Errorf("%s is not a standard library package", path)
	}
	if frag == "" {
		return nil
	}
	pkg, err := c.std.Import(path)
	if err != nil {
		return err
	}
	name, member, isMember := strings.Cut(frag, ".")
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		return fmt.Errorf("%s has no %s", path, name)
	}
	if isMember {
		if _, ok := obj.(*types.TypeName); !ok {
			return fmt.Errorf("%s.%s is not a type", path, name)
		}
		if m, _, _ := types.LookupFieldOrMethod(obj.Type(), true, pkg, member); m == nil {
			return fmt.Errorf("%s.%s has no field or method %s", path, name, member)
		}
	}
	return nil
}

// check runs on one page's source.
func (c *stdChecker) check(page, src string) error {
	var problems []string
	for _, m := range docLinkRE.FindAllStringSubmatch(src, -1) {
		if err := c.checkDocLink(m[1], m[2]); err != nil {
			problems = append(problems, "bad documentation link: "+err.Error())
		}
	}
	for _, block := range exerciseBlocks(src) {
		syms, err := c.answerSymbols(block)
		if err != nil {
			return fmt.Errorf("%s: %q: %w", page, blockTitle(block), err)
		}
		for _, key := range syms {
			if !c.seen[key] && !linked(block, key) {
				path, name, _ := strings.Cut(key, "#")
				problems = append(problems, fmt.Sprintf("%q uses %s.%s for the first time without linking https://pkg.go.dev/%s#%s",
					blockTitle(block), path, name, path, name))
			}
			// Extra tasks may be skipped, so they do not count as having
			// introduced a symbol to the chapters that follow.
			if !strings.HasPrefix(blockTitle(block), "Extra") {
				c.seen[key] = true
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s: standard library documentation:\n  %s", page, strings.Join(problems, "\n  "))
}

// methodKey names a method of a standard library type by the type the
// expression was written against, which is where the documentation lists it
// even when the method is promoted from an unexported embedded type (as
// testing.T.Errorf is). It returns "" for anything else.
func methodKey(sel *types.Selection) string {
	if sel == nil || (sel.Kind() != types.MethodVal && sel.Kind() != types.MethodExpr) {
		return ""
	}
	t := sel.Recv()
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := t.(*types.Named)
	if !ok || n.Obj().Pkg() == nil || !n.Obj().Exported() || !sel.Obj().Exported() || !isStdPath(n.Obj().Pkg().Path()) {
		return ""
	}
	return n.Obj().Pkg().Path() + "#" + n.Obj().Name() + "." + sel.Obj().Name()
}
