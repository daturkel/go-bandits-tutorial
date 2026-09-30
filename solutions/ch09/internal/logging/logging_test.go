package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(&buf, "json", slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("hello", "n", 3)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("output %q is not JSON: %v", buf.String(), err)
	}
	if rec["msg"] != "hello" || rec["level"] != "INFO" || rec["n"] != float64(3) {
		t.Errorf("record = %v", rec)
	}
}

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	log, _ := New(&buf, "text", slog.LevelInfo)
	log.Info("hello", "n", 3)
	if out := buf.String(); !strings.Contains(out, "msg=hello") || !strings.Contains(out, "n=3") {
		t.Errorf("output = %q", out)
	}
}

func TestLevelFilters(t *testing.T) {
	var buf bytes.Buffer
	log, _ := New(&buf, "text", slog.LevelWarn)
	log.Info("quiet")
	log.Debug("quieter")
	log.Warn("loud")
	out := buf.String()
	if strings.Contains(out, "quiet") || !strings.Contains(out, "loud") {
		t.Errorf("output = %q, want only the warning", out)
	}
}

func TestUnknownFormat(t *testing.T) {
	if _, err := New(&bytes.Buffer{}, "xml", slog.LevelInfo); err == nil {
		t.Error("expected an error for an unknown format")
	}
}
