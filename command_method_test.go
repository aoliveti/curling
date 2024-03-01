package curling

import (
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/url"
	"testing"
)

func Test_NewFromRequest_methods(t *testing.T) {
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
			name: "short empty method",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: nil,
			},
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'POST' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'PATCH' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'HEAD' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'PUT' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'DELETE' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'OPTIONS' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'CONNECT' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'TRACE' 'https://localhost/test'",
				},
			},
			wantErr: false,
		},
		{
			name: "long empty method",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
				opts: []Option{WithLongForm()},
			},
			want: &Command{
				tokens: []string{
					"curl --request 'GET' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'GET' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'POST' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'PATCH' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'HEAD' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'PUT' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'DELETE' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'OPTIONS' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'CONNECT' 'https://localhost/test'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'TRACE' 'https://localhost/test'",
				},
				useLongForm: true,
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
