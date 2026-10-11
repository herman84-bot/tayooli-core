package config

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

// ProductionAppURL is the canonical public URL of Tayooli ERP.
const ProductionAppURL = "https://tayooli.my.id"

// DevAppURL is used only outside production when APP_URL is unset.
const DevAppURL = "http://localhost:3000"

// ResolveAppURL is the single source of the public base URL used in links
// sent to users (password reset, email verification, invitations, payment
// return URLs). It trims a trailing slash and never yields a localhost URL in
// production: an unset, invalid or loopback APP_URL there is logged loudly
// and replaced by ProductionAppURL.
func ResolveAppURL(appEnv, raw string) string {
	v := strings.TrimRight(strings.TrimSpace(raw), "/")
	prod := appEnv == "production"
	if !prod {
		if v == "" {
			return DevAppURL
		}
		return v
	}
	if err := validatePublicURL(v); err != nil {
		log.Printf("[CONFIG] ERROR: APP_URL %v in production; email links will use %s. Set APP_URL=%s on the backend service.",
			err, ProductionAppURL, ProductionAppURL)
		return ProductionAppURL
	}
	return v
}

func validatePublicURL(v string) error {
	if v == "" {
		return fmt.Errorf("is not set")
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%q is not an absolute URL", v)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("%q must use https", v)
	}
	h := strings.ToLower(u.Hostname())
	if h == "localhost" || h == "127.0.0.1" || h == "0.0.0.0" || h == "::1" || strings.HasSuffix(h, ".localhost") {
		return fmt.Errorf("%q points to localhost", v)
	}
	return nil
}

// BuildAppLink joins path and an already-encoded query onto the base URL,
// equivalent to new URL(path+"?"+query, base) in JS.
func BuildAppLink(base, path, rawQuery string) (string, error) {
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid app URL %q", base)
	}
	ref := &url.URL{Path: path, RawQuery: rawQuery}
	return u.ResolveReference(ref).String(), nil
}
