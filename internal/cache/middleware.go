package cache

import (
	"net/http"
	"net/http/httptest"
	"time"
)

//nolint:errcheck
func Middleware(storage Storage, duration string, handler func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := storage.Get(r.RequestURI)
		if content != nil {
			w.Write(content)
		} else {
			c := httptest.NewRecorder()
			handler(c, r)

			for k, v := range c.Result().Header {
				w.Header()[k] = v
			}

			w.WriteHeader(c.Code)
			content := c.Body.Bytes()

			if d, err := time.ParseDuration(duration); err == nil {
				storage.Set(r.RequestURI, content, d)
			}

			w.Write(content)
		}
	})
}
