package curling

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errReader struct{}

func (e *errReader) Read(_ []byte) (n int, err error) {
	return 0, assert.AnError
}

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
			name: "body with Content-Length header",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(body)),
					Header: http.Header{"Content-Length": {strconv.Itoa(len(body))}},
				},
			},
			want:    "curl --data-raw 'key=value' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "masked body option",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("sensitive-secret-data")),
				},
				opts: []Option{WithMaskedBody()},
			},
			want:    "curl --data-raw '[CONTENT MASKED]' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "masked body priority over max size",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("very-long-sensitive-data")),
				},
				opts: []Option{WithMaskedBody(), WithMaxBodySize(5)},
			},
			want:    "curl --data-raw '[CONTENT MASKED]' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "masked body with nil body",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   nil,
				},
				opts: []Option{WithMaskedBody()},
			},
			want:    "curl -X 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "masked body with http.NoBody",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   http.NoBody,
				},
				opts: []Option{WithMaskedBody()},
			},
			want:    "curl -X 'POST' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "masked body with empty string reader",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("")),
				},
				opts: []Option{WithMaskedBody()},
			},
			want:    "curl --data-raw '[CONTENT MASKED]' 'https://localhost/test'",
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
			name: "short form body with fallback body size",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(body)),
				},
				opts: []Option{WithMaxBodySize(0)},
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
		{
			name: "json masking success",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`{"user": "admin", "pass": "secret", "meta": {"token": "123"}}`)),
				},
				opts: []Option{WithMaskedJSONFields("pass", "token")},
			},
			want:    `curl --data-raw '{"meta":{"token":"*****"},"pass":"*****","user":"admin"}' 'https://localhost/test'`,
			wantErr: assert.NoError,
		},
		{
			name: "json masking inside array",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`[{"id": 1, "token": "secret1"}, {"id": 2, "token": "secret2"}]`)),
				},
				opts: []Option{WithMaskedJSONFields("token")},
			},
			want:    `curl --data-raw '[{"id":1,"token":"*****"},{"id":2,"token":"*****"}]' 'https://localhost/test'`,
			wantErr: assert.NoError,
		},
		{
			name: "json masking top-level object",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`{"id": 101, "billing": {"card": "4444", "address": {"city": "Rome"}}}`)),
				},
				opts: []Option{WithMaskedJSONFields("billing")},
			},
			want:    `curl --data-raw '{"billing":"*****","id":101}' 'https://localhost/test'`,
			wantErr: assert.NoError,
		},
		{
			name: "json masking fast check failure (non-json body)",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("plain text body starting with non-json char")),
				},
				opts: []Option{WithMaskedJSONFields("password")},
			},
			want:    "curl --data-raw '[CONTENT MASKED: invalid or truncated JSON]' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "json masking fail-secure on invalid json",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`{"user": "admin", "pass": "sec`)),
				},
				opts: []Option{WithMaskedJSONFields("pass")},
			},
			want:    "curl --data-raw '[CONTENT MASKED: invalid or truncated JSON]' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "json masking fail-secure on truncation",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`{"user": "admin", "pass": "sec"}`)),
				},
				opts: []Option{WithMaskedJSONFields("pass"), WithMaxBodySize(5)},
			},
			want:    "curl --data-raw '[CONTENT MASKED: invalid or truncated JSON]' 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "body with image/jpeg content type is omitted",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("binary-jpeg-data")),
					Header: http.Header{"Content-Type": {"image/jpeg"}},
				},
			},
			want:    "curl --data-raw '[BINARY DATA OMITTED: image/jpeg]' 'https://localhost/test' -H 'Content-Type: image/jpeg'",
			wantErr: assert.NoError,
		},
		{
			name: "body with application/octet-stream content type is omitted",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("raw-binary-stream")),
					Header: http.Header{"Content-Type": {"application/octet-stream"}},
				},
			},
			want:    "curl --data-raw '[BINARY DATA OMITTED: application/octet-stream]' 'https://localhost/test' -H 'Content-Type: application/octet-stream'",
			wantErr: assert.NoError,
		},
		{
			name: "body with application/zip content type is omitted",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("zip-archive")),
					Header: http.Header{"Content-Type": {"application/zip"}},
				},
			},
			want:    "curl --data-raw '[BINARY DATA OMITTED: application/zip]' 'https://localhost/test' -H 'Content-Type: application/zip'",
			wantErr: assert.NoError,
		},
		{
			name: "body with text/plain content type is not omitted",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("text data")),
					Header: http.Header{"Content-Type": {"text/plain"}},
				},
			},
			want:    "curl --data-raw 'text data' 'https://localhost/test' -H 'Content-Type: text/plain'",
			wantErr: assert.NoError,
		},
		{
			name: "body with Content-Encoding gzip is omitted, overriding Content-Type",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`{"status": "ok"}`)),
					Header: http.Header{"Content-Type": {"application/json"}, "Content-Encoding": {"gzip"}},
				},
			},
			want:    "curl --data-raw '[ENCODED DATA OMITTED: Content-Encoding: gzip]' 'https://localhost/test' -H 'Content-Encoding: gzip' -H 'Content-Type: application/json'",
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

func TestNewFromRequest_GetBody_Success(t *testing.T) {
	t.Parallel()

	payload := "fast-path-data"
	req, err := http.NewRequest(http.MethodPost, "https://localhost/test", strings.NewReader(payload))
	require.NoError(t, err)

	cmd, err := NewFromRequest(req)
	require.NoError(t, err)
	assert.Contains(t, cmd.String(), fmt.Sprintf("--data-raw '%s'", payload))
}

func TestNewFromRequest_GetBody_Truncation(t *testing.T) {
	t.Parallel()
	payload := "1234567890"
	req, err := http.NewRequest(http.MethodPost, "https://localhost/test", strings.NewReader(payload))
	require.NoError(t, err)

	cmd, err := NewFromRequest(req, WithMaxBodySize(5))
	require.NoError(t, err)
	assert.Contains(t, cmd.String(), "--data-raw '12345... (truncated body, total 10 bytes)'")
}

func TestNewFromRequest_GetBody_Failure_Fallback(t *testing.T) {
	t.Parallel()
	payload := "data-via-fallback"
	req, err := http.NewRequest(http.MethodPost, "https://localhost/test", strings.NewReader(payload))
	require.NoError(t, err)

	req.GetBody = func() (io.ReadCloser, error) {
		return nil, assert.AnError
	}

	cmd, err := NewFromRequest(req)
	require.NoError(t, err)
	assert.Contains(t, cmd.String(), fmt.Sprintf("--data-raw '%s'", payload))

	restored, _ := io.ReadAll(req.Body)
	assert.Equal(t, payload, string(restored))
}

func TestNewFromRequest_GetBody_ReadError(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequest(http.MethodPost, "https://localhost/test", strings.NewReader("ok"))
	require.NoError(t, err)

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(&errReader{}), nil
	}

	cmd, err := NewFromRequest(req)
	require.Nil(t, err)
	assert.Contains(t, cmd.String(), "--data-raw 'ok'")
}

func TestNewFromRequest_BodyRestoration(t *testing.T) {
	t.Parallel()

	opts := []Option{WithMaxBodySize(10)}

	testUrl := &url.URL{
		Scheme: "https",
		Host:   "localhost",
	}

	tests := []struct {
		name         string
		originalBody []byte
	}{
		{
			name:         "body smaller than limit",
			originalBody: []byte("12345"), // 5 bytes < 10 byte limit
		},
		{
			name:         "body equal to limit",
			originalBody: []byte("1234567890"), // 10 bytes == 10 byte limit
		},
		{
			name:         "body larger than limit (truncation)",
			originalBody: []byte("12345678901234"), // 14 bytes > 10 byte limit
		},
		{
			name:         "empty body",
			originalBody: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &http.Request{
				Method: http.MethodPost,
				URL:    testUrl,
				Body:   io.NopCloser(bytes.NewReader(tt.originalBody)),
			}

			_, err := NewFromRequest(r, opts...)
			require.NoError(t, err, "NewFromRequest should not fail")

			restoredBody, err := io.ReadAll(r.Body)
			require.NoError(t, err, "Failed to read the restored body")

			assert.Equal(t, tt.originalBody, restoredBody, "Body content was not restored correctly")
		})
	}
}

func TestNewFromRequest_BodyRestoration_Masked(t *testing.T) {
	t.Parallel()

	testUrl := &url.URL{
		Scheme: "https",
		Host:   "localhost",
	}
	originalData := []byte("sensitive-data")

	r := &http.Request{
		Method: http.MethodPost,
		URL:    testUrl,
		Body:   io.NopCloser(bytes.NewReader(originalData)),
	}

	_, err := NewFromRequest(r, WithMaskedBody())
	require.NoError(t, err)

	restoredBody, err := io.ReadAll(r.Body)
	require.NoError(t, err)

	assert.Equal(t, originalData, restoredBody)
}
