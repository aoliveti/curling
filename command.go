package curling

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// A Command represents a cURL command.
type Command struct {
	// tokens holds the complete lines of the command.
	tokens []string

	// cfg holds all user-configurable settings.
	cfg config

	// model is the pre-processed request data used by the builders.
	model parsedRequest
}

// parsedRequest holds pre-calculated data from the *http.Request.
type parsedRequest struct {
	request *http.Request

	hasAuth bool
	user    string
	pass    string

	hasData bool
	body    *bytes.Buffer

	hasCookies bool
	// cookies is the formatted string of all cookies (e.g., "k1=v1; k2=v2").
	cookies string

	// bodyWasTruncated is true if the body exceeded maxBodySize.
	bodyWasTruncated bool
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

	if err := c.model.build(r, c.cfg); err != nil {
		return nil, err
	}

	c.construct()

	return &c, nil
}

// build preprocesses the *http.Request into the internal parsedRequest.
func (m *parsedRequest) build(r *http.Request, cfg config) error {
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
	buf := new(bytes.Buffer)

	var rr io.Reader = r.Body
	if cfg.maxBodySize > 0 {
		// If a limit is set, wrap the original body in a LimitReader.
		// rr now reads from r.Body but will stop after maxBodySize bytes.
		rr = io.LimitReader(r.Body, cfg.maxBodySize)
	}

	bytesRead, err := io.Copy(buf, rr)
	if err != nil {
		return fmt.Errorf("error reading request body: %w", err)
	}

	// If a limit was set and we read exactly that many bytes,
	// we must check if the original body still has data.
	if cfg.maxBodySize > 0 && bytesRead == cfg.maxBodySize {
		// Try to read one more byte from the original r.Body.
		// If n > 0, the LimitReader stopped us, so truncation occurred.
		if n, _ := r.Body.Read(make([]byte, 1)); n > 0 {
			m.bodyWasTruncated = true
		}
	}

	// Restore the original body
	r.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))

	m.body = buf
	m.hasData = true

	return nil
}

// construct is the internal orchestrator.
// It runs all the small autonomous builder functions in order.
func (c *Command) construct() {
	// handledHeaders tracks headers handled by builders (e.g., Auth)
	handledHeaders := make(map[string]bool)

	commandParts := []string{"curl"}
	commandParts = buildOptions(commandParts, c.cfg)
	commandParts = buildAuth(commandParts, c.cfg, c.model, handledHeaders)
	commandParts = buildCookies(commandParts, c.cfg, c.model, handledHeaders)
	commandParts = buildData(commandParts, c.cfg, c.model)
	commandParts = buildMethod(commandParts, c.cfg, c.model)
	commandParts = buildURL(commandParts, c.cfg, c.model)

	headerParts := buildHeaders(c.cfg, c.model, handledHeaders)

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

// buildAuth adds the -u/--user flag and handle the Authorization header.
func buildAuth(args []string, cfg config, model parsedRequest, handledHeaders map[string]bool) []string {
	if !model.hasAuth {
		return args
	}

	authStr := fmt.Sprintf("%s:%s", model.user, model.pass)
	args = append(args, optionForm(cfg.style, "-u", "--user"), escape(cfg.style, authStr))
	handledHeaders["Authorization"] = true

	return args
}

// buildCookies adds the -b/--cookie flag and handle the Cookie header.
func buildCookies(args []string, cfg config, model parsedRequest, handledHeaders map[string]bool) []string {
	if !model.hasCookies {
		return args
	}

	args = append(args, optionForm(cfg.style, "-b", "--cookie"), escape(cfg.style, model.cookies))
	handledHeaders["Cookie"] = true

	return args
}

// buildData adds the --data-raw flag if data exists.
func buildData(args []string, cfg config, model parsedRequest) []string {
	// We only add the flag if a body was present (even if empty).
	if model.body == nil {
		return args
	}

	body := model.body.String()

	// Add the marker if the body was truncated
	if model.bodyWasTruncated {
		if model.contentLength > 0 {
			body += fmt.Sprintf("... (truncated body, total %d bytes)", model.contentLength)
		} else {
			body += "... (truncated body)"
		}
	}

	return append(args, "--data-raw", escape(cfg.style, body))
}

// buildMethod adds the -X flag if it is not a cURL default.
func buildMethod(args []string, cfg config, model parsedRequest) []string {
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
func buildURL(args []string, cfg config, model parsedRequest) []string {
	return append(args, escape(cfg.style, model.request.URL.String()))
}

// buildHeaders builds all non-handled HTTP headers.
func buildHeaders(cfg config, model parsedRequest, handledHeaders map[string]bool) []string {
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
