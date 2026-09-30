package ex2

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func lastRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &rec); err != nil {
		t.Fatalf("output %q is not JSON: %v", buf.String(), err)
	}
	return rec
}

func TestRedactsListedKeys(t *testing.T) {
	var buf bytes.Buffer
	log := NewRedactingLogger(&buf, "password", "token")
	log.Info("login", "user", "ada", "password", "hunter2", "token", "abc")
	rec := lastRecord(t, &buf)

	if rec["user"] != "ada" || rec["msg"] != "login" {
		t.Errorf("other fields must be untouched: %v", rec)
	}
	if rec["password"] != "[REDACTED]" || rec["token"] != "[REDACTED]" {
		t.Errorf("secrets not redacted: %v", rec)
	}
	if strings.Contains(buf.String(), "hunter2") {
		t.Error("the secret value appears in the output")
	}
}

func TestRedactsInsideGroupsAndWith(t *testing.T) {
	var buf bytes.Buffer
	log := NewRedactingLogger(&buf, "token")

	log.Info("grouped", slog.Group("auth", "user", "ada", "token", "abc"))
	rec := lastRecord(t, &buf)
	auth, _ := rec["auth"].(map[string]any)
	if auth["token"] != "[REDACTED]" || auth["user"] != "ada" {
		t.Errorf("group attribute: %v", rec)
	}

	buf.Reset()
	log.With("token", "abc").Info("with")
	if rec := lastRecord(t, &buf); rec["token"] != "[REDACTED]" {
		t.Errorf("With attribute: %v", rec)
	}
}

func TestNoKeysMeansNoRedaction(t *testing.T) {
	var buf bytes.Buffer
	NewRedactingLogger(&buf).Info("plain", "password", "visible")
	if rec := lastRecord(t, &buf); rec["password"] != "visible" {
		t.Errorf("record = %v", rec)
	}
}

func TestBuiltInAttributesSurvive(t *testing.T) {
	var buf bytes.Buffer
	NewRedactingLogger(&buf, "msg-not-a-key").Warn("careful")
	rec := lastRecord(t, &buf)
	if rec["level"] != "WARN" || rec["msg"] != "careful" || rec["time"] == nil {
		t.Errorf("standard fields lost: %v", rec)
	}
}
