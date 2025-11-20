package curling

import (
	"bufio"
	"bytes"
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
	c.cfg.maskedHeaders = make(map[string]struct{})
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

	// If there is no body, we're done.
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
func buildOptions(args []string, cfg config, data requestData) []string {
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
	if data.body == nil {
		return args
	}

	body := data.body.String()

	// Add the marker if the body was truncated
	if data.bodyTruncated {
		if data.contentLength > 0 {
			body += fmt.Sprintf("... (truncated body, total %d bytes)", data.contentLength)
		} else {
			body += "... (truncated body)"
		}
	}

	return append(args, "--data-raw", escape(cfg.style, body))
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
		args = append(args, optionForm(cfg.style, "-X", "--request"), escape(cfg.style, method))
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
			// Check if the header is configured to be masked.
			// If so, replace the actual value with a placeholder.
			if isMasked(cfg, canonicalKey) {
				value = "*****"
			}

			headers = append(headers, fmt.Sprintf("%s: %s", canonicalKey, value))
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

// isMasked checks if the given header key is in the maskedHeaders map.
func isMasked(cfg config, key string) bool {
	_, ok := cfg.maskedHeaders[http.CanonicalHeaderKey(key)]
	return ok
}

// escape sanitizes the string based on the configured shell type.
func escape(style outputStyle, s string) string {
	switch style.shell {
	case shellPowerShell:
		return escapePowerShell(style, s)
	case shellWindowsCMD:
		return escapeWindowsCMD(s)
	default:
		return escapeUnix(style, s)
	}
}

// escapeUnix handles escaping for Bash/Zsh/Unix shells.
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
func escapePowerShell(style outputStyle, s string) string {
	if style.useDoubleQuotes {
		// PowerShell Double Quotes: escape ` " $
		// Note: Backslash is NOT escaped in PS strings usually, but Backtick is.
		v := strings.ReplaceAll(s, "`", "``")
		v = strings.ReplaceAll(v, "\"", "`\"")
		v = strings.ReplaceAll(v, "$", "`$")
		return fmt.Sprintf("\"%s\"", v)
	}

	// PowerShell Single Quotes: escape ' as ''
	v := strings.ReplaceAll(s, "'", "''")
	return fmt.Sprintf("'%s'", v)
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
