package config

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// env builds a fake getenv from key=value pairs.
func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestDefaults(t *testing.T) {
	cfg, err := Load(nil, env(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		Addr: "localhost:8080", Policy: "thompson", Arms: 3, Seed: 1,
		LogLevel: slog.LevelInfo, LogFormat: "text", ShutdownGrace: 10 * time.Second,
		PendingTTL: 10 * time.Minute, ExpireInterval: 30 * time.Second,
	}
	if cfg != want {
		t.Errorf("defaults = %+v, want %+v", cfg, want)
	}
}

func TestPrecedence(t *testing.T) {
	// Environment beats the default.
	cfg, err := Load(nil, env("BANDIT_ARMS", "7", "BANDIT_LOG_LEVEL", "debug", "BANDIT_SHUTDOWN_GRACE", "3s"), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Arms != 7 || cfg.LogLevel != slog.LevelDebug || cfg.ShutdownGrace != 3*time.Second {
		t.Errorf("env values not applied: %+v", cfg)
	}
	// A flag beats the environment.
	cfg, err = Load([]string{"-arms", "9", "-log-level", "error"}, env("BANDIT_ARMS", "7", "BANDIT_LOG_LEVEL", "debug"), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Arms != 9 || cfg.LogLevel != slog.LevelError {
		t.Errorf("flags did not override env: %+v", cfg)
	}
}

func TestInvalidValuesAreAllReported(t *testing.T) {
	_, err := Load([]string{"-arms", "0"},
		env("BANDIT_SEED", "banana", "BANDIT_SHUTDOWN_GRACE", "soon", "BANDIT_LOG_FORMAT", "xml"), io.Discard)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
	msg := err.Error()
	for _, want := range []string{"BANDIT_SEED", "BANDIT_SHUTDOWN_GRACE", "arms must be at least 1", "log format"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %q", msg, want)
		}
	}
}

func TestBadLogLevel(t *testing.T) {
	_, err := Load([]string{"-log-level", "chatty"}, env(), io.Discard)
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("error = %v, want ErrInvalid", err)
	}
}

func TestHelp(t *testing.T) {
	var out strings.Builder
	_, err := Load([]string{"-h"}, env("BANDIT_ARMS", "5"), &out)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want flag.ErrHelp", err)
	}
	// Usage shows the environment variable for each flag, and current defaults.
	if u := out.String(); !strings.Contains(u, "BANDIT_ARMS") || !strings.Contains(u, "(default 5)") {
		t.Errorf("usage text lacks env names or env-derived defaults:\n%s", u)
	}
}

func TestUnknownFlag(t *testing.T) {
	if _, err := Load([]string{"-bogus"}, env(), io.Discard); err == nil {
		t.Error("expected an error for an unknown flag")
	}
}

func TestStorageSettings(t *testing.T) {
	cfg, err := Load([]string{"-expire-reward", "0"},
		env("BANDIT_DATABASE_URL", "postgres://db/bandits", "BANDIT_PENDING_TTL", "90s"), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://db/bandits" || cfg.PendingTTL != 90*time.Second {
		t.Errorf("storage settings not applied: %+v", cfg)
	}
	if cfg.ExpireReward == nil || *cfg.ExpireReward != 0 {
		t.Errorf("ExpireReward = %v, want a pointer to 0 (an explicit zero, not unset)", cfg.ExpireReward)
	}

	cfg, err = Load(nil, env(), io.Discard)
	if err != nil || cfg.ExpireReward != nil {
		t.Errorf("unset ExpireReward = (%v, %v), want nil", cfg.ExpireReward, err)
	}
}

func TestBadStorageSettings(t *testing.T) {
	for _, args := range [][]string{
		{"-expire-reward", "2"},
		{"-expire-reward", "lots"},
		{"-pending-ttl", "0s"},
		{"-expire-interval", "-1s"},
	} {
		if _, err := Load(args, env(), io.Discard); !errors.Is(err, ErrInvalid) {
			t.Errorf("%v: error = %v, want ErrInvalid", args, err)
		}
	}
}
