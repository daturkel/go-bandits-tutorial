package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGolden pins the exact output of a seeded run. If a change alters the
// numbers, the diff shows up in review; run with -update to accept it.
func TestGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{"run", "-scenario", "easy", "-steps", "5000", "-seed", "42"}, &buf); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "easy_seed42.golden")
	if *update {
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("output differs from %s:\n--- got ---\n%s--- want ---\n%s", path, buf.Bytes(), want)
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"unknown command", []string{"walk"}},
		{"unknown scenario", []string{"run", "-scenario", "nope"}},
		{"unknown policy", []string{"run", "-policy", "nope"}},
		{"bad flag", []string{"run", "-bogus"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.args, &bytes.Buffer{})
			if !errors.Is(err, errUsage) {
				t.Fatalf("error = %v, want a usage error", err)
			}
		})
	}
}

func TestCompareWritesFiles(t *testing.T) {
	dir := t.TempDir()
	csvPath, svgPath := filepath.Join(dir, "r.csv"), filepath.Join(dir, "r.svg")
	var buf bytes.Buffer
	args := []string{"compare", "-scenario", "easy", "-steps", "300", "-seeds", "4", "-csv", csvPath, "-svg", svgPath}
	if err := run(args, &buf); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{csvPath, svgPath} {
		if info, err := os.Stat(p); err != nil || info.Size() == 0 {
			t.Errorf("%s not written: %v", p, err)
		}
	}
	if !bytes.Contains(buf.Bytes(), []byte("thompson")) {
		t.Errorf("summary missing thompson:\n%s", buf.String())
	}
}
