package curling

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type readCloser struct {
	io.Reader
	io.Closer
}

// A Command represents a cURL command.
type Command struct {
	// tokens holds the complete lines of the command.
	tokens []string

	// cfg holds all user-configurable settings.
	cfg config

	// data contains the details extracted from the *http.Request.
	data requestData
}

// requestData holds pre-calculated data from the *http.Request.
type requestData struct {
	request *http.Request

	hasAuth bool
	user    string
	pass    string

	hasData bool
	body    *bytes.Buffer

	hasCookies bool
	// cookies is the formatted string of all cookies (e.g., "k1=v1; k2=v2").
	cookies string

	// bodyTruncated is true if the body exceeded maxBodySize.
	bodyTruncated bool
	// contentLength holds the original Content-Length header, if present.
	contentLength int64
}

// NewFromRequest returns a new [Command] that reads from r.
func NewFromRequest(r *http.Request, opts ...Option) (*Command, error) {
	var c Command

	// Set default config values
	c.cfg.maxBodySize = defaultMaxBodySize

	for _, opt := range opts {
		opt(&c)
	}

	if r.URL == nil {
		return nil, fmt.Errorf("request url is nil")
	}

	if err := c.data.load(r, c.cfg); err != nil {
		return nil, err
	}

	c.compile()

	return &c, nil
}

// load extracts relevant data from the *http.Request and populates the internal model.
// It performs a non-destructive read (peek) of the body and restores it,
// ensuring the request remains valid for subsequent handlers.
func (m *requestData) load(r *http.Request, cfg config) error {
	m.request = r
	m.user, m.pass, m.hasAuth = r.BasicAuth()
	// Store the original content length
	m.contentLength = r.ContentLength

	// Pre-parse cookies
	cookies := r.Cookies()
	if len(cookies) > 0 {
		m.hasCookies = true
		var cookieParts []string
		for _, cookie := range cookies {
			cookieParts = append(cookieParts, cookie.String())
		}
		m.cookies = strings.Join(cookieParts, "; ")
	}

	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}

	// Create the buffer that will hold the body
	peekSize := cfg.maxBodySize

	// Wrap the original body in a bufio.Reader.
	// This is essential for non-destructive peeking.
	b := bufio.NewReaderSize(r.Body, peekSize+1)

	// Peek(peekSize + 1) is the key to detecting truncation.
	// We try to read one byte more than the limit.
	peekBuffer, err := b.Peek(peekSize + 1)

	// Only hard I/O errors are fatal.
	// We must allow io.EOF (body < peekSize)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("error reading request body: %w", err)
	}

	m.body = bytes.NewBuffer(peekBuffer)
	m.hasData = true

	// Check for truncation.
	// We peeked one byte past the limit (peekSize + 1).
	// If the error is not EOF, it means the body is longer than peekSize.
	if !errors.Is(err, io.EOF) {
		m.bodyTruncated = true
		// Cut the log buffer down to the exact peekSize.
		m.body.Truncate(peekSize)
	}

	// Restore the full request body for subsequent handlers.
	r.Body = &readCloser{
		Reader: b,
		Closer: r.Body,
	}

	return nil
}

// compile assembles the final cURL command tokens.
// It executes the specific builder functions (headers, auth, body, etc.)
// and aggregates their output into the final command structure.
func (c *Command) compile() {
	// handledHeaders tracks headers handled by builders (e.g., Auth)
	handledHeaders := make(map[string]bool)

	commandParts := []string{"curl"}
	commandParts = buildOptions(commandParts, c.cfg)
	commandParts = buildAuth(commandParts, c.cfg, c.data, handledHeaders)
	commandParts = buildCookies(commandParts, c.cfg, c.data, handledHeaders)
	commandParts = buildData(commandParts, c.cfg, c.data)
	commandParts = buildMethod(commandParts, c.cfg, c.data)
	commandParts = buildURL(commandParts, c.cfg, c.data)

	headerParts := buildHeaders(c.cfg, c.data, handledHeaders)

	c.tokens = assembleTokens(commandParts, headerParts)
}

// String returns the cURL command.
func (c *Command) String() string {
	separator := " "
	if c.cfg.style.useMultiLine {
		separator = fmt.Sprintf(" %s\n", c.cfg.style.lineContinuation)
	}

	s := strings.Join(c.tokens, separator)
	return strings.TrimSpace(s)
}

// buildOptions adds basic curl flags (-s, -k, -L, -m, --compressed)
func buildOptions(args []string, cfg config) []string {
	if cfg.flags.silent {
		args = append(args, optionForm(cfg.style, "-s", "--silent"))
	}
	if cfg.requestTimeout > 0 {
		args = append(args, optionForm(cfg.style, "-m", "--max-time"), strconv.Itoa(cfg.requestTimeout))
	}
	if cfg.flags.insecure {
		args = append(args, optionForm(cfg.style, "-k", "--insecure"))
	}
	if cfg.flags.compressed {
		args = append(args, "--compressed")
	}
	if cfg.flags.location {
		args = append(args, optionForm(cfg.style, "-L", "--location"))
	}
	return args
}

// buildAuth adds the -u/--user flag and handles the Authorization header.
func buildAuth(args []string, cfg config, model requestData, handledHeaders map[string]bool) []string {
	if !model.hasAuth {
		return args
	}

	authStr := fmt.Sprintf("%s:%s", model.user, model.pass)
	args = append(args, optionForm(cfg.style, "-u", "--user"), escape(cfg.style, authStr))
	handledHeaders["Authorization"] = true

	return args
}

// buildCookies adds the -b/--cookie flag and handles the Cookie header.
func buildCookies(args []string, cfg config, model requestData, handledHeaders map[string]bool) []string {
	if !model.hasCookies {
		return args
	}

	args = append(args, optionForm(cfg.style, "-b", "--cookie"), escape(cfg.style, model.cookies))
	handledHeaders["Cookie"] = true

	return args
}

// buildData adds the --data-raw flag if data exists.
func buildData(args []string, cfg config, model requestData) []string {
	// We only add the flag if a body was present (even if empty).
	if model.body == nil {
		return args
	}

	body := model.body.String()

	// Add the marker if the body was truncated
	if model.bodyTruncated {
		if model.contentLength > 0 {
			body += fmt.Sprintf("... (truncated body, total %d bytes)", model.contentLength)
		} else {
			body += "... (truncated body)"
		}
	}

	return append(args, "--data-raw", escape(cfg.style, body))
}

// buildMethod adds the -X flag if it is not a cURL default.
func buildMethod(args []string, cfg config, model requestData) []string {
	method := model.request.Method
	if method == "" {
		if model.hasData {
			method = http.MethodPost
		} else {
			method = http.MethodGet
		}
	}

	isGetDefault := method == http.MethodGet && !model.hasData
	isPostDefault := method == http.MethodPost && model.hasData

	if !isGetDefault && !isPostDefault {
		args = append(args, optionForm(cfg.style, "-X", "--request"), escape(cfg.style, method))
	}

	return args
}

// buildURL escapes and adds the URL to the end of the main args.
func buildURL(args []string, cfg config, model requestData) []string {
	return append(args, escape(cfg.style, model.request.URL.String()))
}

// buildHeaders builds all non-handled HTTP headers.
func buildHeaders(cfg config, model requestData, handledHeaders map[string]bool) []string {
	r := model.request
	if len(r.Header) == 0 && r.Host == "" {
		return nil
	}

	host := r.Host
	var headers []string
	var headerTokens []string

	for key, values := range r.Header {
		canonicalKey := http.CanonicalHeaderKey(key)

		if handledHeaders[canonicalKey] {
			continue
		}

		if canonicalKey == "Host" {
			if host == "" {
				host = strings.Join(values, ", ")
			}
			continue
		}
		headers = append(headers, fmt.Sprintf("%s: %s", canonicalKey, strings.Join(values, ", ")))
	}

	if host != "" {
		headers = append(headers, fmt.Sprintf("Host: %s", host))
	}

	slices.Sort(headers)

	for _, header := range headers {
		h := strings.Join([]string{optionForm(cfg.style, "-H", "--header"), escape(cfg.style, header)}, " ")
		headerTokens = append(headerTokens, h)
	}

	return headerTokens
}

// assembleTokens constructs the final c.tokens slice.
func assembleTokens(mainArgs, headerArgs []string) []string {
	mainCmd := strings.Join(mainArgs, " ")
	tokens := []string{mainCmd}
	tokens = append(tokens, headerArgs...)
	return tokens
}

// optionForm returns the correct form based on config.
func optionForm(style outputStyle, short, long string) string {
	if style.useLongForm {
		return long
	}
	return short
}

// escape escapes a string based on config.
func escape(style outputStyle, s string) string {
	if style.useDoubleQuotes {
		v := strings.ReplaceAll(s, "\"", "\\\"")
		v = strings.ReplaceAll(v, "`", "\\`")
		v = strings.ReplaceAll(v, "$", "\\$")
		return fmt.Sprintf("\"%s\"", v)
	}

	v := strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", v)
}
