package config

import (
	"net/url"
	"strings"
	"testing"
)

func TestResolveAppURL(t *testing.T) {
	cases := []struct {
		env, raw, want string
	}{
		{"production", "https://tayooli.my.id/", "https://tayooli.my.id"},
		{"production", "", ProductionAppURL},
		{"production", "http://localhost:3000", ProductionAppURL},
		{"production", "https://127.0.0.1", ProductionAppURL},
		{"production", "http://tayooli.my.id", ProductionAppURL},
		{"production", "not a url", ProductionAppURL},
		{"production", "https://tayooli-frontend.zeabur.app", "https://tayooli-frontend.zeabur.app"},
		{"development", "", DevAppURL},
		{"development", "http://localhost:3000/", "http://localhost:3000"},
	}
	for _, c := range cases {
		if got := ResolveAppURL(c.env, c.raw); got != c.want {
			t.Errorf("ResolveAppURL(%q,%q) = %q, want %q", c.env, c.raw, got, c.want)
		}
	}
}

func TestResolveAppURL_ProductionNeverLocalhost(t *testing.T) {
	for _, raw := range []string{"", "http://localhost:3000", "https://localhost", "https://app.localhost", "http://[::1]:3000"} {
		if got := ResolveAppURL("production", raw); strings.Contains(got, "localhost") || strings.Contains(got, "::1") {
			t.Errorf("production APP_URL %q resolved to %q", raw, got)
		}
	}
}

func TestBuildAppLink(t *testing.T) {
	got, err := BuildAppLink("https://tayooli.my.id/", "/reset-password", "token="+url.QueryEscape("a b&c"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://tayooli.my.id/reset-password?token=a+b%26c"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if _, err := BuildAppLink("", "/x", ""); err == nil {
		t.Error("empty base must error")
	}
}
