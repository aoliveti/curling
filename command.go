package curling

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
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

	uri *url.URL

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
	c.cfg.style.lineContinuation = lineContinuationDefault
	c.cfg.style.shell = POSIX

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
func (d *requestData) load(r *http.Request, cfg config) error {
	d.request = r
	d.user, d.pass, d.hasAuth = r.BasicAuth()
	// Store the original content length
	d.contentLength = r.ContentLength

	// Pre-parse cookies
	cookies := r.Cookies()
	if len(cookies) > 0 {
		d.hasCookies = true

		// Sort cookies by name for deterministic output.
		sort.Slice(cookies, func(i, j int) bool {
			return cookies[i].Name < cookies[j].Name
		})

		var cookieParts []string
		for _, cookie := range cookies {
			cookieParts = append(cookieParts, cookie.String())
		}
		d.cookies = strings.Join(cookieParts, "; ")
	}

	// We create a shallow copy of the URL structure.
	// This allows us to modify the Scheme/Host in our local copy (d.uri)
	// without mutating the original URL which might be shared.
	u := *r.URL
	d.uri = &u

	// In a server (middleware) context, r.URL.Scheme and r.URL.Host are often missing.
	if d.uri.Scheme == "" || d.uri.Host == "" {
		d.uri.Scheme = "http"

		// Check Proxy Headers (if trusted)
		if cfg.flags.trustProxy {
			if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
				// Handle multiple hops (e.g. "https, http").
				// We only care about the client's original protocol (the first one).
				if comma := strings.Index(proto, ","); comma != -1 {
					proto = strings.TrimSpace(proto[:comma])
				}
				d.uri.Scheme = proto
			}
		}

		// TLS check overrides "http", but usually we respect X-Forwarded-Proto if present
		if d.uri.Scheme == "http" && r.TLS != nil {
			d.uri.Scheme = "https"
		}

		// If URL.Host is empty, fallback to the Request.Host (header value).
		if d.uri.Host == "" && r.Host != "" {
			d.uri.Host = r.Host
		}
	}

	// If there is no Body, we're done.
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}

	// If body masking is enabled, we skip reading the body entirely.
	if cfg.maskBody {
		d.hasData = true
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

	d.body = bytes.NewBuffer(peekBuffer)
	d.hasData = true

	// Check for truncation.
	// We peeked one byte past the limit (peekSize + 1).
	// If the error is not EOF, it means the body is longer than peekSize.
	if !errors.Is(err, io.EOF) {
		d.bodyTruncated = true
		// Cut the log buffer down to the exact peekSize.
		d.body.Truncate(peekSize)
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
	commandParts = buildOptions(commandParts, c.cfg, c.data)
	commandParts = buildAuth(commandParts, c.cfg, c.data, handledHeaders)
	commandParts = buildCookies(commandParts, c.cfg, c.data, handledHeaders)
	commandParts = buildData(commandParts, c.cfg, c.data)
	commandParts = buildMethod(commandParts, c.cfg, c.data)
	commandParts = buildURL(commandParts, c.cfg, c.data)

	headerParts := buildHeaders(c.cfg, c.data, handledHeaders)

	tokens := make([]string, 0, len(commandParts)+len(headerParts))
	tokens = append(tokens, commandParts...)
	tokens = append(tokens, headerParts...)
	c.tokens = tokens
}

// String returns the cURL command.
func (c *Command) String() string {
	if len(c.tokens) == 0 {
		return ""
	}

	// Single-line mode (default): just join tokens with spaces.
	if !c.cfg.style.useMultiLine {
		return strings.Join(c.tokens, " ")
	}

	// Multi-line mode: use continuation character and indentation.
	var sb strings.Builder
	// separator: space + continuation char + newline + 2 spaces indentation
	sep := fmt.Sprintf(" %s\n  ", c.cfg.style.lineContinuation)

	// The first token (curl) acts as the anchor, no indentation prefix.
	sb.WriteString(c.tokens[0])

	// Subsequent tokens get the indented separator prefix.
	for _, token := range c.tokens[1:] {
		sb.WriteString(sep)
		sb.WriteString(token)
	}

	return sb.String()
}

// buildOptions adds basic curl flags (-s, -k, -L, -m, --compressed)
func buildOptions(args []string, cfg config, data requestData) []string {
	if cfg.flags.silent {
		args = append(args, optionForm(cfg.style, "-s", "--silent"))
	}
	if cfg.requestTimeout > 0 {
		flag := optionForm(cfg.style, "-m", "--max-time")
		args = append(args, fmt.Sprintf("%s %s", flag, strconv.Itoa(cfg.requestTimeout)))
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

	// Smart Globbing:
	// If the URL contains characters that cURL interprets as globbing ranges ([] or {}),
	// we automatically disable globbing to treat the URL literally.
	// This prevents syntax errors with IPv6 addresses (e.g., [::1]) or query parameters
	// containing brackets (e.g., filters[id]=1).
	if strings.ContainsAny(data.uri.String(), "[]{}") {
		args = append(args, optionForm(cfg.style, "-g", "--globoff"))
	}

	return args
}

// buildAuth adds the -u/--user flag and handles the Authorization header.
func buildAuth(args []string, cfg config, data requestData, handledHeaders map[string]bool) []string {
	if !data.hasAuth {
		return args
	}

	authValue := fmt.Sprintf("%s:%s", data.user, data.pass)

	// If Authorization is masked, we specifically redact the password part
	// of the basic auth credential string to prevent leakage in the -u flag.
	if isMasked(cfg, "Authorization") {
		authValue = fmt.Sprintf("%s:*****", data.user)
	}

	args = append(args, optionForm(cfg.style, "-u", "--user"), escape(cfg.style, authValue))
	handledHeaders["Authorization"] = true

	return args
}

// buildCookies adds the -b/--cookie flag and handles the Cookie header.
func buildCookies(args []string, cfg config, data requestData, handledHeaders map[string]bool) []string {
	if !data.hasCookies {
		return args
	}

	args = append(args, optionForm(cfg.style, "-b", "--cookie"), escape(cfg.style, data.cookies))
	handledHeaders["Cookie"] = true

	return args
}

// buildData adds the --data-raw flag if data exists.
func buildData(args []string, cfg config, data requestData) []string {
	// We only add the flag if a body was present (even if empty).
	if !data.hasData {
		return args
	}

	appendData := func(value string) []string {
		return append(args, fmt.Sprintf("--data-raw %s", escape(cfg.style, value)))
	}

	if cfg.maskBody {
		return appendData("[CONTENT MASKED]")
	}

	body := data.body.String()

	if len(cfg.maskedJSONFields) > 0 {
		if data.bodyTruncated {
			return appendData("[CONTENT MASKED: invalid or truncated JSON]")
		}

		maskedBody, err := maskJSONContent(body, cfg.maskedJSONFields)
		if err != nil {
			return appendData("[CONTENT MASKED: invalid or truncated JSON]")
		}

		return appendData(maskedBody)
	}

	// Add the marker if the body was truncated
	if data.bodyTruncated {
		if data.contentLength > 0 {
			body += fmt.Sprintf("... (truncated body, total %d bytes)", data.contentLength)
		} else {
			body += "... (truncated body)"
		}
	}

	return appendData(body)
}

// buildMethod adds the -X flag if it is not a cURL default.
func buildMethod(args []string, cfg config, data requestData) []string {
	method := data.request.Method
	if method == "" {
		if data.hasData {
			method = http.MethodPost
		} else {
			method = http.MethodGet
		}
	}

	isGetDefault := method == http.MethodGet && !data.hasData
	isPostDefault := method == http.MethodPost && data.hasData

	if !isGetDefault && !isPostDefault {
		flag := optionForm(cfg.style, "-X", "--request")
		val := escape(cfg.style, method)
		args = append(args, fmt.Sprintf("%s %s", flag, val))
	}

	return args
}

// buildURL escapes and adds the URL to the end of the main args.
func buildURL(args []string, cfg config, data requestData) []string {
	return append(args, escape(cfg.style, data.uri.String()))
}

// buildHeaders builds all non-handled HTTP headers.
func buildHeaders(cfg config, data requestData, handledHeaders map[string]bool) []string {
	r := data.request
	if len(r.Header) == 0 && r.Host == "" {
		return nil
	}

	// The Host header is usually promoted to the Request.Host field
	// and removed from the Header map. We start with that value.
	host := r.Host
	var headers []string
	var headerTokens []string

	for key, values := range r.Header {
		canonicalKey := http.CanonicalHeaderKey(key)

		// We remove Content-Length because cURL calculates it automatically based on the body data.
		// Keeping the original header is dangerous if we truncated the body (mismatch)
		if canonicalKey == "Content-Length" {
			continue
		}

		// We skip headers that builders already handled (Auth, Cookies).
		if handledHeaders[canonicalKey] {
			continue
		}

		// Handle Host header extraction separately.
		if canonicalKey == "Host" {
			if host == "" && len(values) > 0 {
				// RFC 7230: A client MUST send a Host header field in all HTTP/1.1 request messages.
				// We take the first value as the authoritative host for comparison.
				host = values[0]
			}
			continue
		}

		// Iterate over all values for this header key.
		// This ensures that headers with multiple values are represented as multiple -H flags,
		// avoiding data corruption if values contain commas.
		for _, value := range values {
			sv := sanitizeHeaderValue(cfg, canonicalKey, value)
			headers = append(headers, fmt.Sprintf("%s: %s", canonicalKey, sv))
		}
	}

	// Host: We only add it explicitly if it differs from the host in the calculated URL.
	// This avoids redundant "-H 'Host: ...'" flags when cURL would infer it automatically.
	if host != "" && data.uri.Host != host {
		headers = append(headers, fmt.Sprintf("Host: %s", host))
	}

	slices.Sort(headers)

	for _, header := range headers {
		h := strings.Join([]string{optionForm(cfg.style, "-H", "--header"), escape(cfg.style, header)}, " ")
		headerTokens = append(headerTokens, h)
	}

	return headerTokens
}

// optionForm returns the correct form based on config.
func optionForm(style outputStyle, short, long string) string {
	if style.useLongForm {
		return long
	}
	return short
}

// sanitizeHeaderValue resolves the final string representation of a header value by evaluating configuration rules.
// It first checks if an environment variable substitution is defined, which takes the highest priority.
// If no substitution is found, it checks if the header is configured to be masked and returns a redacted placeholder.
// If neither condition is met, it returns the original header value unchanged.
func sanitizeHeaderValue(cfg config, key, value string) string {
	if envVar, ok := cfg.envSubstitutions[key]; ok {
		return envVar
	}

	if isMasked(cfg, key) {
		return "*****"
	}

	return value
}

// isMasked checks if the given header key is in the maskedHeaders map.
func isMasked(cfg config, key string) bool {
	_, ok := cfg.maskedHeaders[http.CanonicalHeaderKey(key)]
	return ok
}

// maskJSONContent parses the body and redacts specific keys recursively.
// Returns an error if the body is not valid JSON.
func maskJSONContent(body string, fields map[string]struct{}) (string, error) {
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return "", fmt.Errorf("not a valid JSON object or array")
	}

	var v interface{}
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return "", err
	}

	maskJSONFields(v, fields)

	maskedBody, err := json.Marshal(v)
	return string(maskedBody), err
}

// maskJSONFields traverses the interface{} tree to find and mask the JSON fields.
func maskJSONFields(data interface{}, fields map[string]struct{}) {
	switch d := data.(type) {
	case map[string]interface{}:
		for k, v := range d {
			if _, ok := fields[k]; ok {
				d[k] = "*****"
				continue
			}
			maskJSONFields(v, fields)
		}
	case []interface{}:
		for _, v := range d {
			maskJSONFields(v, fields)
		}
	}
}

// escape sanitizes the string based on the configured shell type.
func escape(style outputStyle, s string) string {
	switch style.shell {
	case PowerShell:
		return escapePowerShell(s)
	case WindowsCMD:
		return escapeWindowsCMD(s)
	default:
		return escapeUnix(style, s)
	}
}

// escapeUnix handles escaping for Bash/Zsh/POSIX shells.
func escapeUnix(style outputStyle, s string) string {
	if style.useDoubleQuotes {
		// Bash Double Quotes: escape \ " ` $
		v := strings.ReplaceAll(s, "\\", "\\\\")
		v = strings.ReplaceAll(v, "\"", "\\\"")
		v = strings.ReplaceAll(v, "`", "\\`")
		v = strings.ReplaceAll(v, "$", "\\$")
		return fmt.Sprintf("\"%s\"", v)
	}

	// Bash Single Quotes: strictly literal, escape ' as '\''
	v := strings.ReplaceAll(s, "'", "'\\''")
	return fmt.Sprintf("'%s'", v)
}

// escapePowerShell handles escaping for PowerShell.
// We enforce double quotes to safely handle variables and special characters using backticks.
func escapePowerShell(s string) string {
	// PowerShell Double Quotes Rules:
	// - Escape backtick (`) with another backtick (``).
	// - Escape double quote (") with backtick (`").
	// - Escape variable start ($) with backtick (`$).
	v := strings.ReplaceAll(s, "`", "``")
	v = strings.ReplaceAll(v, "\"", "`\"")
	v = strings.ReplaceAll(v, "$", "`$")
	return fmt.Sprintf("\"%s\"", v)
}

// escapeWindowsCMD handles escaping for Windows Command Prompt (cmd.exe).
// It follows Microsoft C Runtime parsing rules.
// Arguments are wrapped in double quotes.
// Backslashes are literal, unless they precede a double quote.
// If they precede a double quote, they must be doubled.
// Double quotes are escaped as \".
// Trailing backslashes (preceding the closing quote) must be doubled.
func escapeWindowsCMD(s string) string {
	if s == "" {
		return "\"\""
	}

	var sb strings.Builder
	// Pre-allocate: original length + 2 quotes + margin for escapes
	sb.Grow(len(s) + 16)
	sb.WriteByte('"') // Opening quote

	backslashCount := 0
	for _, c := range s {
		switch c {
		case '\\':
			backslashCount++
			continue

		case '"':
			// Backslashes before a quote must be doubled (N -> 2N)
			if backslashCount > 0 {
				sb.WriteString(strings.Repeat("\\", backslashCount*2))
			}
			backslashCount = 0
			// Escape the quote itself
			sb.WriteString(`\"`)
			continue
		}

		// Default case: Normal character
		// Flush pending backslashes literally (N -> N)
		if backslashCount > 0 {
			sb.WriteString(strings.Repeat("\\", backslashCount))
		}
		backslashCount = 0
		sb.WriteRune(c)
	}

	// Handle trailing backslashes (at the end of the string).
	// They must be doubled because they technically precede the closing quote.
	if backslashCount > 0 {
		sb.WriteString(strings.Repeat("\\", backslashCount*2))
	}

	sb.WriteByte('"') // Closing quote
	return sb.String()
}
