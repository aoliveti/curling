package curling

import (
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Test_NewFromRequest_body(t *testing.T) {
	testUrl := &url.URL{
		Scheme: "https",
		Host:   "localhost",
		Path:   "test",
	}

	body := "key=value"
	r, err := http.NewRequest(http.MethodPost, testUrl.String(), strings.NewReader(body))
	if err != nil {
		t.Errorf("new request: %v", err)
		return
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
			name: "short form no body",
			args: args{
				r: &http.Request{
					URL:    testUrl,
					Method: http.MethodPost,
				},
			},
			want: &Command{
				tokens: []string{
					"curl -X 'POST' 'https://localhost/test'",
				},
			},
			wantErr: false,
		},
		{
			name: "short form body",
			args: args{
				r: r,
			},
			want: &Command{
				tokens: []string{
					"curl -X 'POST' 'https://localhost/test'",
					"-d 'key=value'",
				},
			},
			wantErr: false,
		},
		{
			name: "long form no body",
			args: args{
				r: &http.Request{
					URL:    testUrl,
					Method: http.MethodPost,
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
			name: "long form body",
			args: args{
				r:    r,
				opts: []Option{WithLongForm()},
			},
			want: &Command{
				tokens: []string{
					"curl --request 'POST' 'https://localhost/test'",
					"--data 'key=value'",
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
