package main

import (
	"go/scanner"
	"go/token"
	"html"
	"strings"
)

// seg is a run of source text with a CSS class ("" means plain).
type seg struct {
	class string
	text  string
}

var builtins = map[string]bool{
	"bool": true, "byte": true, "complex64": true, "complex128": true, "error": true,
	"float32": true, "float64": true, "int": true, "int8": true, "int16": true,
	"int32": true, "int64": true, "rune": true, "string": true, "uint": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"any": true, "comparable": true, "true": true, "false": true, "nil": true, "iota": true,
	"append": true, "cap": true, "clear": true, "close": true, "copy": true, "delete": true,
	"len": true, "make": true, "max": true, "min": true, "new": true, "panic": true,
	"print": true, "println": true, "recover": true,
}

type gotok struct {
	tok  token.Token
	lit  string
	text string
	off  int
}

// highlightGo splits Go source into classed segments. Whitespace between
// tokens is preserved exactly by copying the gaps from the source.
func highlightGo(src string) []seg {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, []byte(src), nil, scanner.ScanComments)

	var toks []gotok
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON && lit == "\n" {
			continue // inserted by the scanner; no source text
		}
		text := lit
		if text == "" {
			text = tok.String()
		}
		toks = append(toks, gotok{tok: tok, lit: lit, text: text, off: file.Offset(pos)})
	}

	// Mark identifiers that define a function/type or are called.
	class := make([]string, len(toks))
	for i, t := range toks {
		switch {
		case t.tok == token.COMMENT:
			class[i] = "c"
		case t.tok == token.STRING || t.tok == token.CHAR:
			class[i] = "s"
		case t.tok == token.INT || t.tok == token.FLOAT || t.tok == token.IMAG:
			class[i] = "n"
		case t.tok.IsKeyword():
			class[i] = "k"
		case t.tok == token.IDENT:
			switch {
			case builtins[t.lit]:
				class[i] = "b"
			case i+1 < len(toks) && toks[i+1].tok == token.LPAREN:
				class[i] = "f"
			}
		}
	}
	for i, t := range toks {
		if t.tok == token.TYPE && i+1 < len(toks) && toks[i+1].tok == token.IDENT {
			class[i+1] = "d"
		}
		if t.tok == token.FUNC && i+1 < len(toks) {
			j := i + 1
			if toks[j].tok == token.LPAREN {
				// Either a method receiver, `func (r T) Name(`, or a function
				// literal/type. Skip the parenthesis group and see whether a
				// name followed by `(` or `[` comes next.
				depth := 0
				for ; j < len(toks); j++ {
					if toks[j].tok == token.LPAREN {
						depth++
					} else if toks[j].tok == token.RPAREN {
						depth--
						if depth == 0 {
							j++
							break
						}
					}
				}
				if !(j+1 < len(toks) && toks[j].tok == token.IDENT &&
					(toks[j+1].tok == token.LPAREN || toks[j+1].tok == token.LBRACK)) {
					continue
				}
			}
			if j < len(toks) && toks[j].tok == token.IDENT && class[j] != "b" {
				class[j] = "d"
			}
		}
	}

	var out []seg
	prev := 0
	for i, t := range toks {
		if t.off > prev {
			out = append(out, seg{"", src[prev:t.off]})
		}
		out = append(out, seg{class[i], src[t.off : t.off+len(t.text)]})
		prev = t.off + len(t.text)
	}
	if prev < len(src) {
		out = append(out, seg{"", src[prev:]})
	}
	return out
}

// highlightPlain handles non-Go files: `#` and `//` comments (`--` for SQL,
// selected by isSQL) and quoted strings.
func highlightPlain(src string, isSQL bool) []seg {
	var out []seg
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			out = append(out, seg{"", plain.String()})
			plain.Reset()
		}
	}
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case (!isSQL && c == '#') || (c == '/' && i+1 < len(src) && src[i+1] == '/' && !isSQL) ||
			(isSQL && c == '-' && i+1 < len(src) && src[i+1] == '-'):
			j := strings.IndexByte(src[i:], '\n')
			if j < 0 {
				j = len(src) - i
			}
			flush()
			out = append(out, seg{"c", src[i : i+j]})
			i += j
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(src) && src[j] != c && src[j] != '\n' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j < len(src) && src[j] == c {
				j++
			}
			if j > len(src) {
				j = len(src)
			}
			flush()
			out = append(out, seg{"s", src[i:j]})
			i = j
		default:
			plain.WriteByte(c)
			i++
		}
	}
	flush()
	return out
}

// splitLines turns segments into per-line HTML fragments, closing and
// reopening spans at line breaks so every line is self-contained.
func splitLines(segs []seg) []string {
	var lines []string
	var cur strings.Builder
	emit := func(class, text string) {
		if text == "" {
			return
		}
		if class == "" {
			cur.WriteString(html.EscapeString(text))
			return
		}
		cur.WriteString(`<span class="` + class + `">` + html.EscapeString(text) + `</span>`)
	}
	for _, sg := range segs {
		parts := strings.Split(sg.text, "\n")
		for i, part := range parts {
			if i > 0 {
				lines = append(lines, cur.String())
				cur.Reset()
			}
			emit(sg.class, strings.TrimSuffix(part, "\r"))
		}
	}
	lines = append(lines, cur.String())
	return lines
}
