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
	if err := run([]string{"-scenario", "easy", "-steps", "5000", "-seed", "42"}, &buf); err != nil {
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
		{"unknown scenario", []string{"-scenario", "nope"}},
		{"unknown policy", []string{"-policy", "nope"}},
		{"bad flag", []string{"-bogus"}},
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
