package curling

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

type readerWithError struct{}

func (r readerWithError) Close() error {
	return nil
}

// Read attempts to read data into the provided byte slice but always returns an error.
func (r readerWithError) Read(p []byte) (n int, err error) {
	_ = p
	return 0, fmt.Errorf("error reading data")
}

func TestCommand_String(t *testing.T) {
	t.Parallel()

	type fields struct {
		tokens []string
		cfg    config
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "empty tokens slice",
			fields: fields{
				tokens: nil,
			},
			want: "",
		},
		{
			name: "one token",
			fields: fields{
				tokens: []string{"a"},
			},
			want: "a",
		},
		{
			name: "two tokens",
			fields: fields{
				tokens: []string{"a", "b"},
			},
			want: "a b",
		},
		{
			name: "multiline",
			fields: fields{
				tokens: []string{
					"curl -X 'POST' 'https://localhost/test'",
					"-H 'X-Key-1: 1'",
					"-d 'key=value'",
				},
				cfg: config{
					style: outputStyle{
						useMultiLine:     true,
						lineContinuation: lineContinuationDefault,
					},
				},
			},
			want: "curl -X 'POST' 'https://localhost/test' \\\n-H 'X-Key-1: 1' \\\n-d 'key=value'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Command{
				tokens: tt.fields.tokens,
				cfg:    tt.fields.cfg,
			}

			if got := c.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_NewFromRequest(t *testing.T) {
	t.Parallel()

	type args struct {
		r    *http.Request
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *Command
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "invalid url",
			args: args{
				r: &http.Request{
					URL: nil,
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "error reading body",
			args: args{
				r: &http.Request{
					URL: &url.URL{
						Scheme: "https",
						Host:   "localhost",
					},
					Body: readerWithError{},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewFromRequest(tt.args.r, tt.args.opts...)

			if !tt.wantErr(t, err, "NewFromRequest() error") {
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
