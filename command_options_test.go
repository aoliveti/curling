package curling

import (
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/url"
	"testing"
)

func Test_NewFromRequest_options(t *testing.T) {
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
		want    *Command
		wantErr bool
	}{
		{
			name: "short location option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithFollowRedirects()},
			},
			want: &Command{
				tokens: []string{
					"curl -L -X 'GET' 'https://localhost/test'",
				},
				location: true,
			},
			wantErr: false,
		},
		{
			name: "short insecure option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithInsecure()},
			},
			want: &Command{
				tokens: []string{
					"curl -k -X 'GET' 'https://localhost/test'",
				},
				insecure: true,
			},
			wantErr: false,
		},
		{
			name: "short silent option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithSilent()},
			},
			want: &Command{
				tokens: []string{
					"curl -s -X 'GET' 'https://localhost/test'",
				},
				silent: true,
			},
			wantErr: false,
		},
		{
			name: "short request timeout option (positive value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithRequestTimeout(5)},
			},
			want: &Command{
				tokens: []string{
					"curl -m 5 -X 'GET' 'https://localhost/test'",
				},
				requestTimeout: 5,
			},
			wantErr: false,
		},
		{
			name: "short request timeout option (negative value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithRequestTimeout(-5)},
			},
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
				},
				requestTimeout: 0,
			},
			wantErr: false,
		},
		{
			name: "long location option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithFollowRedirects(), WithLongForm()},
			},
			want: &Command{
				tokens: []string{
					"curl --location --request 'GET' 'https://localhost/test'",
				},
				useLongForm: true,
				location:    true,
			},
			wantErr: false,
		},
		{
			name: "long insecure option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithInsecure(), WithLongForm()},
			},
			want: &Command{
				tokens: []string{
					"curl --insecure --request 'GET' 'https://localhost/test'",
				},
				useLongForm: true,
				insecure:    true,
			},
			wantErr: false,
		},
		{
			name: "long silent option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithSilent(), WithLongForm()},
			},
			want: &Command{
				tokens: []string{
					"curl --silent --request 'GET' 'https://localhost/test'",
				},
				useLongForm: true,
				silent:      true,
			},
			wantErr: false,
		},
		{
			name: "long request timeout option (positive value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithLongForm(), WithRequestTimeout(5)},
			},
			want: &Command{
				tokens: []string{
					"curl --max-time 5 --request 'GET' 'https://localhost/test'",
				},
				useLongForm:    true,
				requestTimeout: 5,
			},
			wantErr: false,
		},
		{
			name: "long request timeout option (negative value)",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithLongForm(), WithRequestTimeout(-5)},
			},
			want: &Command{
				tokens: []string{
					"curl --request 'GET' 'https://localhost/test'",
				},
				useLongForm:    true,
				requestTimeout: 0,
			},
			wantErr: false,
		},
		{
			name: "compression option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithCompression()},
			},
			want: &Command{
				tokens: []string{
					"curl --compressed -X 'GET' 'https://localhost/test'",
				},
				compressed: true,
			},
			wantErr: false,
		},
		{
			name: "default multiline option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithMultiLine()},
			},
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
				},
				useMultiLine:     true,
				lineContinuation: lineContinuationDefault,
			},
			wantErr: false,
		},
		{
			name: "windows multiline option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithWindowsMultiLine()},
			},
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
				},
				useMultiLine:     true,
				lineContinuation: lineContinuationWindows,
			},
			wantErr: false,
		},
		{
			name: "powershell multiline option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithPowerShellMultiLine()},
			},
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
				},
				useMultiLine:     true,
				lineContinuation: lineContinuationPowerShell,
			},
			wantErr: false,
		},
		{
			name: "double quotes option",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithDoubleQuotes()},
			},
			want: &Command{
				tokens: []string{
					"curl -X \"GET\" \"https://localhost/test\"",
				},
				useDoubleQuotes: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFromRequest(tt.args.r, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			optUnexported := cmp.AllowUnexported(Command{})
			if !cmp.Equal(got, tt.want, optUnexported) {
				t.Errorf("NewFromRequest() got = %v, want = %v, diff = %v", got, tt.want, cmp.Diff(got, tt.want, optUnexported))
			}
		})
	}
}
