package redirect

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/hysios/mx"
	"github.com/hysios/mx/middleware"
	"github.com/tidwall/gjson"
)

type redirectWriter struct {
	b         bytes.Buffer
	header    http.Header
	stateCode int
}

type Redirector func(redirectUrl string, r *http.Request, w http.ResponseWriter) error

func Middleware(match middleware.Matcher, field string, fn Redirector) mx.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if match(r) {
				var rewriter redirectWriter
				h.ServeHTTP(&rewriter, r)
				rewriter.CopyHeader(w)
				redirctUrl, err := rewriter.Decode(field)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				if err := fn(redirctUrl, r, w); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				h.ServeHTTP(w, r)
			}
		})
	}
}

func (w *redirectWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}

	return w.header
}

func (w *redirectWriter) Write(b []byte) (int, error) {
	return w.b.Write(b)
}

func (w *redirectWriter) WriteHeader(statusCode int) {
	w.stateCode = statusCode
}

func (w *redirectWriter) Bytes() []byte {
	return w.b.Bytes()
}

// Read
func (w *redirectWriter) Read(p []byte) (n int, err error) {
	return w.b.Read(p)
}

// CopyHeader
func (w *redirectWriter) CopyHeader(out http.ResponseWriter) {
	for k, v := range w.Header() {
		for _, vv := range v {
			out.Header().Set(k, vv)
		}
	}
}

// Decode
func (w *redirectWriter) Decode(field string) (string, error) {
	v := gjson.Get(w.b.String(), field)
	if v.Exists() {
		return v.String(), nil
	}
	return "", errors.New("field not found")
}
