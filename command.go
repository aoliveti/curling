package curling

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

type Command struct {
	tokens           []string
	lineContinuation string

	location     bool
	compressed   bool
	insecure     bool
	useLongForm  bool
	useMultiLine bool
}

func NewFromRequest(r *http.Request, opts ...Option) (*Command, error) {
	var c Command

	if err := c.build(r, opts...); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Command) String() string {
	separator := " "
	if c.useMultiLine {
		separator = fmt.Sprintf(" %s\n", c.lineContinuation)
	}

	s := strings.Join(c.tokens, separator)
	return strings.TrimSpace(s)
}

func (c *Command) appendToken(s ...string) {
	token := strings.Join(s, " ")
	c.tokens = append(c.tokens, token)
}

func (c *Command) optionForm(short, long string) string {
	if c.useLongForm {
		return long
	}

	return short
}

func (c *Command) build(r *http.Request, opts ...Option) error {
	for _, opt := range opts {
		opt(c)
	}

	if r.URL == nil {
		return fmt.Errorf("request url is nil")
	}

	c.buildCommand(r)
	c.buildHeaders(r)

	if err := c.buildData(r); err != nil {
		return err
	}

	return nil
}

func (c *Command) buildCommand(r *http.Request) {
	s := []string{"curl"}

	if c.insecure {
		s = append(s, c.optionForm("-k", "--insecure"))
	}

	if c.compressed {
		s = append(s, "--compressed")
	}

	if c.location {
		s = append(s, c.optionForm("-L", "--location"))
	}

	var command string
	if len(s) > 0 {
		command = strings.Join(s, " ")
	}

	method := r.Method
	if method == "" {
		method = http.MethodGet
	}

	c.appendToken(
		command,
		c.optionForm("-X", "--request"),
		escape(method),
		escape(r.URL.String()),
	)
}

func (c *Command) buildHeaders(r *http.Request) {
	if len(r.Header) > 0 {
		var headers []string

		for key, values := range r.Header {
			canonicalKey := http.CanonicalHeaderKey(key)
			headers = append(headers, fmt.Sprintf("%s: %s", canonicalKey, strings.Join(values, ", ")))
		}

		slices.Sort(headers)

		for _, header := range headers {
			c.appendToken(
				c.optionForm("-H", "--header"),
				escape(header),
			)
		}
	}
}

func (c *Command) buildData(r *http.Request) error {
	if r.Body != nil {
		var b bytes.Buffer
		if _, err := b.ReadFrom(r.Body); err != nil {
			return fmt.Errorf("reading bytes from request body: %w", err)
		}

		// Reset request body for potential re-reads
		r.Body = io.NopCloser(bytes.NewBuffer(b.Bytes()))

		option := c.optionForm("-d", "--data")
		c.appendToken(option, escape(b.String()))
	}

	return nil
}

func escape(s string) string {
	v := strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", v)
}
