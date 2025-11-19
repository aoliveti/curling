package curling

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewFromRequest_url(t *testing.T) {
	t.Parallel()

	type args struct {
		r    *http.Request
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "client side request with full url",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Scheme: "https",
						Host:   "api.example.com",
						Path:   "/v1/users",
					},
				},
			},
			want:    "curl 'https://api.example.com/v1/users'",
			wantErr: assert.NoError,
		},
		{
			name: "server side request default http",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					Host:   "localhost:8080",
					URL: &url.URL{
						Path: "/health",
					},
				},
			},
			want:    "curl 'http://localhost:8080/health'",
			wantErr: assert.NoError,
		},
		{
			name: "server side request with tls",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					Host:   "secure.local",
					URL: &url.URL{
						Path: "/login",
					},
					TLS: &tls.ConnectionState{},
				},
			},
			want:    "curl 'https://secure.local/login'",
			wantErr: assert.NoError,
		},
		{
			name: "server side request with query params",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					Host:   "search.com",
					URL: &url.URL{
						Path:     "/find",
						RawQuery: "q=golang&page=1",
					},
				},
			},
			want:    "curl 'http://search.com/find?q=golang&page=1'",
			wantErr: assert.NoError,
		},
		{
			name: "server side request missing leading slash in path",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					Host:   "api.io",
					URL: &url.URL{
						Path: "v2/data",
					},
				},
			},
			want:    "curl 'http://api.io/v2/data'",
			wantErr: assert.NoError,
		},
		{
			name: "proxy header ignored without option",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					Host:   "example.com",
					URL: &url.URL{
						Path: "/proxy",
					},
					Header: http.Header{
						"X-Forwarded-Proto": []string{"https"},
					},
				},
			},
			want:    "curl 'http://example.com/proxy' -H 'X-Forwarded-Proto: https'",
			wantErr: assert.NoError,
		},
		{
			name: "proxy header respected with option",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					Host:   "example.com",
					URL: &url.URL{
						Path: "/proxy",
					},
					Header: http.Header{
						"X-Forwarded-Proto": []string{"https"},
					},
				},
				opts: []Option{WithTrustProxy()},
			},
			want:    "curl 'https://example.com/proxy' -H 'X-Forwarded-Proto: https'",
			wantErr: assert.NoError,
		},
		{
			name: "fallback to url host if request host is empty",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Host: "fallback.com",
						Path: "/test",
					},
				},
			},
			want:    "curl 'http://fallback.com/test'",
			wantErr: assert.NoError,
		},
		{
			name: "ipv6 host",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					Host:   "[::1]:9090",
					URL: &url.URL{
						Path: "/ipv6",
					},
				},
			},
			want:    "curl 'http://[::1]:9090/ipv6'",
			wantErr: assert.NoError,
		},
		{
			name: "complex url with everything",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					Host:   "complex.org",
					URL: &url.URL{
						Path:     "/submit",
						RawQuery: "id=123",
					},
					TLS:  &tls.ConnectionState{},
					Body: io.NopCloser(strings.NewReader(`{"a":1}`)),
				},
			},
			want:    "curl --data-raw '{\"a\":1}' 'https://complex.org/submit?id=123'",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewFromRequest(tt.args.r, tt.args.opts...)

			if !tt.wantErr(t, err, "NewFromRequest() error") {
				return
			}

			assert.Equal(t, tt.want, got.String())
		})
	}
}
