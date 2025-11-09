package curling

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewFromRequest_headers(t *testing.T) {
	t.Parallel()

	testUrl := &url.URL{
		Scheme: "https",
		Host:   "localhost",
		Path:   "test",
	}

	singleValueHeader := http.Header{}
	singleValueHeader.Set("x-key-single", "value 1")

	multiValueHeader := http.Header{}
	multiValueHeader.Set("x-key-multi", "value 1")
	multiValueHeader.Add("x-key-multi", "value 2")

	additionalHeader := http.Header{}
	additionalHeader.Set("x-key-z", "foo")
	additionalHeader.Add("x-key-z", "alpha")
	additionalHeader.Add("x-key-z", "baz")
	additionalHeader.Add("x-key-a", "bar")

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
			name: "short form no headers",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form one header single value",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: singleValueHeader,
				},
			},
			want:    "curl 'https://localhost/test' -H 'X-Key-Single: value 1'",
			wantErr: assert.NoError,
		},
		{
			name: "short form one header multi value",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: multiValueHeader,
				},
			},
			want:    "curl 'https://localhost/test' -H 'X-Key-Multi: value 1, value 2'",
			wantErr: assert.NoError,
		},
		{
			name: "short form multiple sorted headers",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: additionalHeader,
				},
			},
			want:    "curl 'https://localhost/test' -H 'X-Key-A: bar' -H 'X-Key-Z: foo, alpha, baz'",
			wantErr: assert.NoError,
		},
		{
			name: "long form no headers",
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
			name: "long form one header single value",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: singleValueHeader,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl 'https://localhost/test' --header 'X-Key-Single: value 1'",
			wantErr: assert.NoError,
		},
		{
			name: "long form one header multi value",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: multiValueHeader,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl 'https://localhost/test' --header 'X-Key-Multi: value 1, value 2'",
			wantErr: assert.NoError,
		},
		{
			name: "long form multiple sorted headers",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: additionalHeader,
				},
				opts: []Option{WithLongForm()},
			},
			want:    "curl 'https://localhost/test' --header 'X-Key-A: bar' --header 'X-Key-Z: foo, alpha, baz'",
			wantErr: assert.NoError,
		},
		{
			name: "short form r.Host",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Host:   "localhost",
				},
			},
			want:    "curl 'https://localhost/test' -H 'Host: localhost'",
			wantErr: assert.NoError,
		},
		{
			name: "short form r.Host overrides Host header",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Host:   "host-a",
					Header: http.Header{
						"Host": {"host-b"},
					},
				},
			},
			want:    "curl 'https://localhost/test' -H 'Host: host-a'",
			wantErr: assert.NoError,
		},
		{
			name: "short form Host header (r.Host is empty)",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Host:   "",
					Header: http.Header{
						"Host": {"host"},
					},
				},
			},
			want:    "curl 'https://localhost/test' -H 'Host: host'",
			wantErr: assert.NoError,
		},
		{
			name: "short form Authorization header",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: http.Header{
						"Authorization": {"Basic dXNlcjpwYXNz"},
					},
				},
			},
			want:    "curl -u 'user:pass' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form non-canonical header key",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: http.Header{
						"x-lowercase-key": {"value"},
					},
				},
			},
			want:    "curl 'https://localhost/test' -H 'X-Lowercase-Key: value'",
			wantErr: assert.NoError,
		},
		{
			name: "short form cookie",
			args: args{
				r: func() *http.Request {
					r := &http.Request{
						Method: http.MethodGet,
						URL:    testUrl,
						Header: http.Header{},
					}
					r.AddCookie(&http.Cookie{Name: "c1", Value: "v1"})
					return r
				}(),
			},
			want:    "curl -b 'c1=v1' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short form multiple cookies",
			args: args{
				r: func() *http.Request {
					r := &http.Request{
						Method: http.MethodGet,
						URL:    testUrl,
						Header: http.Header{},
					}
					r.AddCookie(&http.Cookie{Name: "c1", Value: "v1"})
					r.AddCookie(&http.Cookie{Name: "c2", Value: "v2"})
					return r
				}(),
			},
			want:    "curl -b 'c1=v1; c2=v2' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long form cookie",
			args: args{
				r: func() *http.Request {
					r := &http.Request{
						Method: http.MethodGet,
						URL:    testUrl,
						Header: http.Header{},
					}
					r.AddCookie(&http.Cookie{Name: "c1", Value: "v1"})
					return r
				}(),
				opts: []Option{WithLongForm()},
			},
			want:    "curl --cookie 'c1=v1' 'https://localhost/test'",
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
