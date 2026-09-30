package ex1

import (
	"io"
	"strings"
	"testing"
)

func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestLoadPrecedence(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  func(string) string
		want Config
	}{
		{"defaults", nil, env(), Config{8080, false}},
		{"env only", nil, env("PORT", "9000", "DEBUG", "true"), Config{9000, true}},
		{"flags only", []string{"-port", "7000", "-debug"}, env(), Config{7000, true}},
		{"flag beats env", []string{"-port", "7000"}, env("PORT", "9000"), Config{7000, false}},
		{"debug flag can turn debug off", []string{"-debug=false"}, env("DEBUG", "true"), Config{8080, false}},
		{"env for one, flag for other", []string{"-debug"}, env("PORT", "9001"), Config{9001, true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Load(tc.args, tc.env, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      func(string) string
		mentions string
	}{
		{"bad PORT", nil, env("PORT", "abc"), "PORT"},
		{"bad DEBUG", nil, env("DEBUG", "maybe"), "DEBUG"},
		{"port too low", []string{"-port", "0"}, env(), "port"},
		{"port too high", nil, env("PORT", "70000"), "port"},
		{"unknown flag", []string{"-nope"}, env(), "nope"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(tc.args, tc.env, io.Discard)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.mentions)) {
				t.Errorf("error %q should mention %q", err, tc.mentions)
			}
		})
	}
}
