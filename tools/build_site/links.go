package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var hrefRE = regexp.MustCompile(`\b(?:href|src)="([^"]*)"`)

// checkLinks verifies that every relative href/src in the built pages points
// at an existing file, and that every #fragment exists as an id in its target.
func checkLinks(root string) error {
	siteDir := filepath.Join(root, "site")
	pages, _ := filepath.Glob(filepath.Join(siteDir, "*.html"))
	sort.Strings(pages)
	ids := map[string]map[string]bool{}
	content := map[string]string{}
	for _, p := range pages {
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		content[p] = string(data)
		set := map[string]bool{}
		for _, m := range idAttrRE.FindAllStringSubmatch(string(data), -1) {
			if set[m[1]] && !strings.Contains(m[1], "{") {
				return fmt.Errorf("%s: duplicate id %q", filepath.Base(p), m[1])
			}
			set[m[1]] = true
		}
		ids[p] = set
	}
	var problems []string
	for _, p := range pages {
		for _, m := range hrefRE.FindAllStringSubmatch(content[p], -1) {
			ref := m[1]
			switch {
			case ref == "", strings.HasPrefix(ref, "http://"), strings.HasPrefix(ref, "https://"),
				strings.HasPrefix(ref, "mailto:"), strings.HasPrefix(ref, "data:"):
				continue
			}
			u, err := url.Parse(ref)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: unparsable link %q", filepath.Base(p), ref))
				continue
			}
			target := p
			if u.Path != "" {
				target = filepath.Join(siteDir, filepath.FromSlash(u.Path))
				if _, err := os.Stat(target); err != nil {
					problems = append(problems, fmt.Sprintf("%s: broken link %q", filepath.Base(p), ref))
					continue
				}
			}
			if u.Fragment != "" {
				set, ok := ids[target]
				if !ok {
					continue // fragment into a non-page file; nothing to verify
				}
				if !set[u.Fragment] {
					problems = append(problems, fmt.Sprintf("%s: link %q: no id %q in %s", filepath.Base(p), ref, u.Fragment, filepath.Base(target)))
				}
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("link check failed:\n  %s", strings.Join(problems, "\n  "))
	}
	fmt.Printf("built %d pages, all internal links resolve\n", len(pages))
	return nil
}
