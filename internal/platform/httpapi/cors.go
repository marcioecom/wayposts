package httpapi

import (
	"net/http"
	"net/url"
	"strings"
)

// CORS allows browser clients from the configured origins. Entries are exact
// origins, "*", or scheme plus wildcard host such as "https://*.workers.dev".
// Bearer auth carries no cookies, so the wildcard does not enable credentialed
// cross-site requests.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make([]string, 0, len(allowedOrigins))
	wildcard := false
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(strings.TrimSuffix(origin, "/"))
		if origin == "" {
			continue
		}
		if origin == "*" {
			wildcard = true
		}
		allowed = append(allowed, origin)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSuffix(r.Header.Get("Origin"), "/")
			if origin == "" {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Add("Vary", "Origin")
			permitted := wildcard || originIn(origin, allowed)
			if permitted {
				if wildcard {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				if !permitted {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func originIn(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		if candidate == origin {
			return true
		}
		if scheme, hostPattern, ok := wildcardOrigin(candidate); ok {
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Scheme != scheme {
				continue
			}
			host := parsed.Hostname()
			if host == hostPattern || strings.HasSuffix(host, "."+hostPattern) {
				return true
			}
		}
	}
	return false
}

func wildcardOrigin(candidate string) (scheme, host string, ok bool) {
	scheme, rest, found := strings.Cut(candidate, "://*.")
	if !found || scheme == "" || rest == "" {
		return "", "", false
	}
	return scheme, rest, true
}
