package server

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// allowedOrigins lists extra origins (comma separated in ALLOWED_ORIGINS)
// allowed to open a websocket, e.g. a frontend dev server.
var allowedOrigins = parseOrigins(os.Getenv("ALLOWED_ORIGINS"))

// adminOrigins lists origins (comma separated in ADMIN_ORIGINS) allowed to
// read the admin API: the web dev server, never the public site.
var adminOrigins = parseOrigins(os.Getenv("ADMIN_ORIGINS"))

func parseOrigins(s string) map[string]bool {
	origins := make(map[string]bool)
	for _, o := range strings.Split(s, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins[o] = true
		}
	}
	return origins
}

// checkOrigin allows same-origin requests and any origin in allowedOrigins.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// not a browser
		return true
	}
	if allowedOrigins[origin] {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}
