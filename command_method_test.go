package curling

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewFromRequest_methods(t *testing.T) {
	t.Parallel()

	testUrl := &url.URL{
		Scheme: "https",
		Host:   "localhost",
		Path:   "test",
	}

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
			name: "short empty method",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: nil,
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short empty method with http.NoBody",
			args: args{
				r: &http.Request{
					URL:  testUrl,
					Body: http.NoBody,
				},
				opts: nil,
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short empty method with body",
			args: args{
				r: &http.Request{
					URL:  testUrl,
					Body: io.NopCloser(bytes.NewReader([]byte("{}"))),
				},
				opts: nil,
			},
			want:    "curl --data-raw '{}' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short get method",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short post method",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short patch method",
			args: args{
				r: &http.Request{
					Method: http.MethodPatch,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'PATCH' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short head method",
			args: args{
				r: &http.Request{
					Method: http.MethodHead,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'HEAD' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short put method",
			args: args{
				r: &http.Request{
					Method: http.MethodPut,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'PUT' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short delete method",
			args: args{
				r: &http.Request{
					Method: http.MethodDelete,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'DELETE' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short options method",
			args: args{
				r: &http.Request{
					Method: http.MethodOptions,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'OPTIONS' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short connect method",
			args: args{
				r: &http.Request{
					Method: http.MethodConnect,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'CONNECT' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short trace method",
			args: args{
				r: &http.Request{
					Method: http.MethodTrace,
					URL:    testUrl,
				},
				opts: nil,
			},
			want:    "curl -X 'TRACE' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long empty method",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long get method",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long post method",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long patch method",
			args: args{
				r: &http.Request{
					Method: http.MethodPatch,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'PATCH' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long head method",
			args: args{
				r: &http.Request{
					Method: http.MethodHead,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'HEAD' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long put method",
			args: args{
				r: &http.Request{
					Method: http.MethodPut,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'PUT' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long delete method",
			args: args{
				r: &http.Request{
					Method: http.MethodDelete,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'DELETE' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long options method",
			args: args{
				r: &http.Request{
					Method: http.MethodOptions,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'OPTIONS' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long connect method",
			args: args{
				r: &http.Request{
					Method: http.MethodConnect,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'CONNECT' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long trace method",
			args: args{
				r: &http.Request{
					Method: http.MethodTrace,
					URL:    testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl --request 'TRACE' 'https://localhost/test'",
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
