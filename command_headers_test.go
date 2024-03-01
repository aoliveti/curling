package curling

import (
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/url"
	"testing"
)

func Test_NewFromRequest_headers(t *testing.T) {
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
		want    *Command
		wantErr bool
	}{
		{
			name: "short form no headers",
			args: args{
				r: &http.Request{
					URL: testUrl,
				},
			},
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
					"-H 'X-Key-Single: value 1'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
					"-H 'X-Key-Multi: value 1, value 2'",
				},
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl -X 'GET' 'https://localhost/test'",
					"-H 'X-Key-A: bar'",
					"-H 'X-Key-Z: foo, alpha, baz'",
				},
			},
			wantErr: false,
		},
		{
			name: "long form no headers",
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
			name: "long form one header single value",
			args: args{
				r: &http.Request{
					Method: http.MethodGet,
					URL:    testUrl,
					Header: singleValueHeader,
				},
				opts: []Option{WithLongForm()},
			},
			want: &Command{
				tokens: []string{
					"curl --request 'GET' 'https://localhost/test'",
					"--header 'X-Key-Single: value 1'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'GET' 'https://localhost/test'",
					"--header 'X-Key-Multi: value 1, value 2'",
				},
				useLongForm: true,
			},
			wantErr: false,
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
			want: &Command{
				tokens: []string{
					"curl --request 'GET' 'https://localhost/test'",
					"--header 'X-Key-A: bar'",
					"--header 'X-Key-Z: foo, alpha, baz'",
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
