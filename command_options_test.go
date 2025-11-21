package curling

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewFromRequest_options(t *testing.T) {
	t.Parallel()

	testUrl := &url.URL{
		Scheme: "https",
		Host:   "localhost",
		Path:   "test",
	}

	specialTestUrl := &url.URL{
		Scheme:   "https",
		Host:     "localhost",
		Path:     "test",
		RawQuery: `q="hello $user"`,
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
			name: "short location option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithFollowRedirects()},
			},
			want:    "curl -L 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short insecure option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithInsecure()},
			},
			want:    "curl -k 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short silent option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithSilent()},
			},
			want:    "curl -s 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short request timeout option (positive value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithRequestTimeout(5)},
			},
			want:    "curl -m 5 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "short request timeout option (negative value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithRequestTimeout(-5)},
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long location option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithFollowRedirects(), WithLongForm()},
			},
			want:    "curl --location 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long insecure option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithInsecure(), WithLongForm()},
			},
			want:    "curl --insecure 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long silent option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithSilent(), WithLongForm()},
			},
			want:    "curl --silent 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long request timeout option (positive value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithLongForm(), WithRequestTimeout(5)},
			},
			want:    "curl --max-time 5 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long request timeout option (negative value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithLongForm(), WithRequestTimeout(-5)},
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "compression option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithCompression()},
			},
			want:    "curl --compressed 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "default multiline option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithMultiLine()},
			},
			want:    "curl 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "bash double quotes with history expansion char (!)",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("Hello!")),
				},
				opts: []Option{WithDoubleQuotes()},
			},
			want:    "curl --data-raw \"Hello!\" \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "windows cmd empty body",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader("")),
				},
				opts: []Option{WithWindowsMultiLine()},
			},
			want:    "curl --data-raw \"\" \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "windows cmd path backslashes",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`C:\Windows\System32`)),
				},
				opts: []Option{WithWindowsMultiLine()},
			},
			want:    "curl --data-raw \"C:\\Windows\\System32\" \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "windows cmd trailing backslash",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`Folder\`)),
				},
				opts: []Option{WithWindowsMultiLine()},
			},
			want:    `curl --data-raw "Folder\\" "https://localhost/test"`,
			wantErr: assert.NoError,
		},
		{
			name: "windows cmd backslash before quote (trigger doubling)",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					Body:   io.NopCloser(strings.NewReader(`path=\"`)),
				},
				opts: []Option{WithWindowsMultiLine()},
			},
			want:    `curl --data-raw "path=\\\"" "https://localhost/test"`,
			wantErr: assert.NoError,
		},
		{
			name: "windows cmd multiple backslashes before quote",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					// Input: Two backslashes before a quote: \\"
					Body: io.NopCloser(strings.NewReader(`Wait\\"`)),
				},
				opts: []Option{WithWindowsMultiLine()},
			},
			want:    `curl --data-raw "Wait\\\\\"" "https://localhost/test"`,
			wantErr: assert.NoError,
		},
		{
			name: "windows multiline option (enforce double quotes)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithWindowsMultiLine()},
			},
			want:    "curl \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "windows multiline option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithWindowsMultiLine(), WithDoubleQuotes()},
			},
			want:    "curl \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "powershell double quotes escaping (default)",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					// Body contains: $cost = "100"
					Body: io.NopCloser(strings.NewReader(`$cost = "100"`)),
				},
				opts: []Option{WithPowerShellMultiLine(), WithDoubleQuotes()},
			},
			// PowerShell uses backtick (`) to escape $ and " inside double-quoted strings
			want:    "curl --data-raw \"`$cost = `\"100`\"\" \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "powershell double quotes escaping (backticks)",
			args: args{
				r: &http.Request{
					Method: http.MethodPost,
					URL:    testUrl,
					// Body contains a literal backtick: cmd `exec`
					Body: io.NopCloser(strings.NewReader("cmd `exec`")),
				},
				opts: []Option{WithPowerShellMultiLine(), WithDoubleQuotes()},
			},
			// The backtick itself must be escaped with another backtick (``)
			want:    "curl --data-raw \"cmd ``exec``\" \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "double quotes option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithDoubleQuotes()},
			},
			want:    "curl \"https://localhost/test\"",
			wantErr: assert.NoError,
		},
		{
			name: "short form multiple options",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithFollowRedirects(), WithInsecure(), WithSilent()},
			},
			want:    "curl -s -k -L 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "long form multiple options",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithFollowRedirects(), WithInsecure(), WithSilent(), WithLongForm()},
			},
			want:    "curl --silent --insecure --location 'https://localhost/test'",
			wantErr: assert.NoError,
		},
		{
			name: "double quotes option with special characters",
			args: args{
				r: &http.Request{
					URL: specialTestUrl,
				},
				opts: []Option{WithDoubleQuotes()},
			},
			want:    "curl \"https://localhost/test?q=\\\"hello \\$user\\\"\"",
			wantErr: assert.NoError,
		},
		{
			name: "kitchen sink (all options enabled)",
			args: args{
				r: &http.Request{
					Method: http.MethodPut,
					URL:    specialTestUrl,
					Host:   "host",
					Header: http.Header{
						"Authorization": {"Basic dXNlcjpwYXNz"},
						"X-Extra":       {"value"},
					},
					Body: io.NopCloser(strings.NewReader("data")),
				},
				opts: []Option{
					WithFollowRedirects(),
					WithCompression(),
					WithInsecure(),
					WithSilent(),
					WithLongForm(),
					WithRequestTimeout(10),
					WithDoubleQuotes(),
				},
			},
			want:    "curl --silent --max-time 10 --insecure --compressed --location --user \"user:pass\" --data-raw \"data\" --request \"PUT\" \"https://localhost/test?q=\\\"hello \\$user\\\"\" --header \"Host: host\" --header \"X-Extra: value\"",
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
