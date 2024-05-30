package upload

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/hysios/mx"
	"github.com/hysios/mx/middleware"
)

type Converter interface {
	Converter(form url.Values, attachment string) (interface{}, error)
}

func Middleware(match middleware.Matcher, fileField string, convert Converter) mx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !match(r) {
				next.ServeHTTP(w, r)
				return
			}

			err := r.ParseForm()
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to parse form: %s", err.Error()), http.StatusBadRequest)
				return
			}

			log.Printf("form %v and %v, %v", r.Form, r.PostForm, r.MultipartForm)

			f, head, err := r.FormFile(fileField)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to get file 'attachment': %s", err.Error()), http.StatusBadRequest)
				return
			}
			defer f.Close()
			// head.Filename
			r.Form.Set("filename", head.Filename)
			r.Form.Set("size", strconv.FormatInt(head.Size, 10))
			out, err := os.CreateTemp("", "upload-")
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to create temp file: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			defer out.Close()

			_, err = io.Copy(out, f)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to copy file: %s", err.Error()), http.StatusInternalServerError)
				return
			}

			req, err := convert.Converter(r.Form, out.Name())
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to convert file: %s", err.Error()), http.StatusInternalServerError)
				return
			}

			var (
				sb  strings.Builder
				enc = json.NewEncoder(&sb)
			)

			if err = enc.Encode(req); err != nil {
				http.Error(w, fmt.Sprintf("failed to encode request: %s", err.Error()), http.StatusInternalServerError)
				return
			}

			r.Body = io.NopCloser(strings.NewReader(sb.String()))

			// do something
			next.ServeHTTP(w, r)
		})
	}
}
