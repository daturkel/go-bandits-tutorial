package ex3

import (
	"slices"
	"strings"
	"testing"
)

func TestParseTargets(t *testing.T) {
	tests := []struct {
		name string
		spec string
		want []string
	}{
		{"one", "policyd:9090", []string{"policyd:9090"}},
		{"several with spaces", " a:1 , b:2,c:3 ", []string{"a:1", "b:2", "c:3"}},
		{"trailing comma", "a:1,b:2,", []string{"a:1", "b:2"}},
		{"ipv6", "[::1]:9090,127.0.0.1:65535", []string{"[::1]:9090", "127.0.0.1:65535"}},
	}
	for _, tc := range tests {
		got, err := ParseTargets(tc.spec)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: ParseTargets(%q) = %v, %v; want %v", tc.name, tc.spec, got, err, tc.want)
		}
	}
}

func TestParseTargetsErrors(t *testing.T) {
	tests := []struct {
		spec    string
		mention string // what the error must name
	}{
		{"", ""},
		{" , ,", ""},
		{"a:1,b", "b"},
		{"a:1,:2", ":2"},
		{"a:1,c:", "c:"},
		{"a:1,d:x", "d:x"},
		{"a:0", "a:0"},
		{"a:65536", "a:65536"},
		{"a:1,b:2,a:1", "a:1"},
		{"::1:9090", "::1:9090"}, // an unbracketed IPv6 address is ambiguous
	}
	for _, tc := range tests {
		got, err := ParseTargets(tc.spec)
		if err == nil {
			t.Errorf("ParseTargets(%q) = %v, want an error", tc.spec, got)
			continue
		}
		if !strings.Contains(err.Error(), tc.mention) {
			t.Errorf("ParseTargets(%q): error %q does not mention %q", tc.spec, err, tc.mention)
		}
	}
}
