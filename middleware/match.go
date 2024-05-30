package middleware

import (
	"net/http"
	"strings"
)

type Matcher func(r *http.Request) bool

func PathPrefix(prefix string) Matcher {
	return func(r *http.Request) bool {
		return strings.HasPrefix(r.URL.Path, prefix)
	}
}

func Path(path string) Matcher {
	return func(r *http.Request) bool {
		return r.URL.Path == path
	}
}
