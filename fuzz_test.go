package curling

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Fuzz_NewFromRequest(f *testing.F) {
	f.Add("GET", "https://example.com", "", "X-Key", "value")
	f.Add("POST", "https://foo.com/q?a='b'", "{'a':1}", "Auth", "123")
	f.Add("PUT", "https://'test'.com", "\"", "Host", "host.com")

	f.Fuzz(func(t *testing.T, method, urlStr, body, headerKey, headerValue string) {
		u, _ := url.Parse(urlStr)
		if u == nil {
			u, _ = url.Parse("https://fallback.com")
		}

		r := &http.Request{
			Method: method,
			URL:    u,
			Header: http.Header{
				headerKey: {headerValue},
			},
			Body: io.NopCloser(strings.NewReader(body)),
		}

		_, _ = NewFromRequest(r)
	})
}
