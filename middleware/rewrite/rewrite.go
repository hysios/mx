package rewrite

import (
	"bytes"
	"io"
	"net/http"

	"github.com/hysios/mx"
	"github.com/hysios/mx/middleware"
)

type rewriteWriter struct {
	b         bytes.Buffer
	header    http.Header
	stateCode int
}

type Rewriter interface {
	Before(r *http.Request, w http.ResponseWriter) error
	After(orir *http.Request, body io.ReadCloser, w http.ResponseWriter) error
}

type (
	RewriteBeforeFunc func(r *http.Request, w http.ResponseWriter) error
	RewriteAfterFunc  func(orir *http.Request, body io.ReadCloser, w http.ResponseWriter) error
	RewriteFunc       func(before RewriteBeforeFunc, after RewriteAfterFunc) error
)

func Middleware(match middleware.Matcher, rewrite Rewriter) mx.Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if match(r) {
				if err := rewrite.Before(r, w); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				var rewriter rewriteWriter

				h.ServeHTTP(&rewriter, r)
				copyHeader(rewriter, w)
				if err := rewrite.After(r, io.NopCloser(&rewriter), w); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				h.ServeHTTP(w, r)
			}
		})
	}
}

func copyHeader(rewriter rewriteWriter, w http.ResponseWriter) {
	for k, v := range rewriter.Header() {
		for _, vv := range v {
			w.Header().Set(k, vv)
		}
	}

	// if rewriter.stateCode == 0 {
	// 	rewriter.stateCode = http.StatusOK
	// }

	// w.WriteHeader(rewriter.stateCode)
}

func (w *rewriteWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}

	return w.header
}

func (w *rewriteWriter) Write(b []byte) (int, error) {
	return w.b.Write(b)
}

func (w *rewriteWriter) WriteHeader(statusCode int) {
	w.stateCode = statusCode
}

func (w *rewriteWriter) Bytes() []byte {
	return w.b.Bytes()
}

// Read
func (w *rewriteWriter) Read(p []byte) (n int, err error) {
	return w.b.Read(p)
}
