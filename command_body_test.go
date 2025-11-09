package curling

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewFromRequest_body(t *testing.T) {
	t.Parallel()

	testUrl := &url.URL{
		Scheme: "https",
		Host:   "localhost",
		Path:   "test",
	}
	body := "key=value"

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
			name: "short form nil body",
			args: args{
				r: &http.Request{
					URL:    testUrl,
					Method: http.MethodPost,
					Body:   nil,
				},
			},
			want:    "curl -X 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form http.NoBody",
			args: args{
				r: &http.Request{
					URL:    testUrl,
					Method: http.MethodPost,
					Body:   http.NoBody,
				},
			},
			want:    "curl -X 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form empty string body",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("")),
				},
			},
			want:    "curl --data-raw '' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form body",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(body)),
				},
			},
			want:    "curl --data-raw 'key=value' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long form nil body",
			args: args{
				r: &http.Request{
					URL:    testUrl,
					Method: http.MethodPost,
					Body:   nil,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long form http.NoBody",
			args: args{
				r: &http.Request{
					URL:    testUrl,
					Method: http.MethodPost,
					Body:   http.NoBody,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long form body",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(body)),
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --data-raw 'key=value' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form GET (default)",
			args: args{
				r: &http.Request{
					URL:    testUrl,
					Method: http.MethodGet,
					Body:   http.NoBody,
				},
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form PUT with body (non-default)",
			args: args{
				r: &http.Request{
					Method: http.MethodPut,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(body)),
				},
			},
			want:    "curl --data-raw 'key=value' -X 'PUT' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "default method with body (should be POST)",
			args: args{
				r: &http.Request{
					Method: "",
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(body)),
				},
			},
			want:    "curl --data-raw 'key=value' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form body smaller than limit",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("abc")),
				},
				opts: []Option{WithMaxBodySize(10)},
			},
			want:    "curl --data-raw 'abc' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form body larger than limit",
			args: args{
				r: &http.Request{
					Method:        http.MethodPost,
					URL:           testUrl,
					Body:          io.NopCloser(strings.NewReader("abcdefghijklmn")),
					ContentLength: 14,
				},
				opts: []Option{WithMaxBodySize(10)},
			},
			want:    "curl --data-raw 'abcdefghij... (truncated body, total 14 bytes)' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long form body larger than limit",
			args: args{
				r: &http.Request{
					Method:        http.MethodPost,
					URL:           testUrl,
					Body:          io.NopCloser(strings.NewReader("abcdefghijklmn")),
					ContentLength: 14,
				},
				opts: []Option{WithMaxBodySize(10), WithLongForm()},
			},
			want:    "curl --data-raw 'abcdefghij... (truncated body, total 14 bytes)' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form body larger than limit (unknown length)",
			args: args{
				r: &http.Request{
					Method:        http.MethodPost,
					URL:           testUrl,
					Body:          io.NopCloser(strings.NewReader("abcdefghijklmn")),
					ContentLength: -1,
				},
				opts: []Option{WithMaxBodySize(10)},
			},
			want:    "curl --data-raw 'abcdefghij... (truncated body)' 'https://localhost/test'",
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
