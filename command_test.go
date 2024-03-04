package curling

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/url"
	"testing"
)

// A readerWithError is a fake reader, so the [Read] method return always an error
type readerWithError struct{}

// Close return always nil
func (r readerWithError) Close() error {
	return nil
}

// Read ignores the value of p and return always an error
func (r readerWithError) Read(p []byte) (n int, err error) {
	_ = p
	return 0, fmt.Errorf("error reading data")
}

func TestCommand_String(t *testing.T) {
	type fields struct {
		tokens           []string
		useMultiLine     bool
		lineContinuation string
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
			name: "two tokens with one empty string",
			fields: fields{
				tokens: []string{"a", ""},
			},
			want: "a",
		},
		{
			name: "two tokens with two empty strings",
			fields: fields{
				tokens: []string{"", ""},
			},
			want: "",
		},
		{
			name: "multiline",
			fields: fields{
				tokens: []string{
					"curl -X 'POST' 'https://localhost/test'",
					"-H 'X-Key-1: 1'",
					"-d 'key=value'",
				},
				useMultiLine:     true,
				lineContinuation: lineContinuationDefault,
			},
			want: "curl -X 'POST' 'https://localhost/test' \\\n-H 'X-Key-1: 1' \\\n-d 'key=value'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				tokens:           tt.fields.tokens,
				useMultiLine:     tt.fields.useMultiLine,
				lineContinuation: tt.fields.lineContinuation,
			}
			if got := c.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommand_optionForm(t *testing.T) {
	type fields struct {
		useLongForm bool
	}
	type args struct {
		short string
		long  string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "use short form",
			fields: fields{
				useLongForm: false,
			},
			args: args{
				short: "-F",
				long:  "--foo",
			},
			want: "-F",
		},
		{
			name: "use long form",
			fields: fields{
				useLongForm: true,
			},
			args: args{
				short: "-F",
				long:  "--foo",
			},
			want: "--foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				useLongForm: tt.fields.useLongForm,
			}
			if got := c.optionForm(tt.args.short, tt.args.long); got != tt.want {
				t.Errorf("optionForm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommand_escape(t *testing.T) {
	type fields struct {
		tokens          []string
		useDoubleQuotes bool
	}
	type args struct {
		s string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "without single quotes",
			args: args{
				s: "",
			},
			want: "''",
		},
		{
			name: "with one single quotes",
			args: args{
				s: "'",
			},
			want: "''\\'''",
		},
		{
			name: "with two single quotes",
			args: args{
				s: "'v'",
			},
			want: "''\\''v'\\'''",
		},
		{
			name: "without double quotes",
			fields: fields{
				useDoubleQuotes: true,
			},
			args: args{
				s: "",
			},
			want: "\"\"",
		},
		{
			name: "with one double quotes",
			fields: fields{
				useDoubleQuotes: true,
			},
			args: args{
				s: "\"",
			},
			want: "\"\\\"\"",
		},
		{
			name: "with two double quotes",
			fields: fields{
				useDoubleQuotes: true,
			},
			args: args{
				s: "\"v\"",
			},
			want: "\"\\\"v\\\"\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				tokens:          tt.fields.tokens,
				useDoubleQuotes: tt.fields.useDoubleQuotes,
			}
			if got := c.escape(tt.args.s); got != tt.want {
				t.Errorf("escape() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_NewFromRequest(t *testing.T) {
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
			name: "invalid url",
			args: args{
				r: &http.Request{
					URL: nil,
				},
			},
			wantErr: true,
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
			wantErr: true,
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
