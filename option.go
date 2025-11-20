package curling

import (
	"net/http"
)

const (
	// lineContinuationDefault is the default line continuation character (Unix-like).
	lineContinuationDefault = "\\"
	// lineContinuationWindows is the line continuation character for Windows CMD.
	lineContinuationWindows = "^"
	// lineContinuationPowerShell is the line continuation character for PowerShell.
	lineContinuationPowerShell = "`"

	// defaultMaxBodySize is the default maximum body size (in bytes).
	defaultMaxBodySize = 1024
)

// config holds all user-configurable settings.
type config struct {
	// style holds formatting-related options.
	style outputStyle
	// flags holds boolean cURL options.
	flags curlFlags
	// requestTimeout enables the option -m, --max-time.
	requestTimeout int
	// maxBodySize is the maximum number of bytes to read from the request body.
	maxBodySize int
	// maskedHeaders holds the list of headers to be masked in the output.
	maskedHeaders map[string]struct{}
}

// outputStyle groups options related to the command's text formatting.
type outputStyle struct {
	useLongForm      bool
	useMultiLine     bool
	useDoubleQuotes  bool
	lineContinuation string
}

// curlFlags groups common boolean cURL flags.
type curlFlags struct {
	location   bool
	compressed bool
	insecure   bool
	silent     bool
	// trustProxy enables reading X-Forwarded-* headers to determine the original scheme/host.
	trustProxy bool
}

// Option defines a functional option for configuring a [Command].
type Option func(c *Command)

// WithFollowRedirects enables the option -L, --location.
func WithFollowRedirects() Option {
	return func(c *Command) {
		c.cfg.flags.location = true
	}
}

// WithCompression enables the option --compressed.
func WithCompression() Option {
	return func(c *Command) {
		c.cfg.flags.compressed = true
	}
}

// WithInsecure enables the option -k, --insecure (skip certificate verification).
func WithInsecure() Option {
	return func(c *Command) {
		c.cfg.flags.insecure = true
	}
}

// WithLongForm enables the long form for cURL options.
// Example: --header instead of -H.
func WithLongForm() Option {
	return func(c *Command) {
		c.cfg.style.useLongForm = true
	}
}

// WithSilent enables the option -s, --silent (suppress progress meter).
func WithSilent() Option {
	return func(c *Command) {
		c.cfg.flags.silent = true
	}
}

// WithMultiLine splits the command across multiple lines.
// The default line continuation character is backslash (\).
func WithMultiLine() Option {
	return func(c *Command) {
		c.cfg.style.useMultiLine = true
		c.cfg.style.lineContinuation = lineContinuationDefault
	}
}

// WithWindowsMultiLine splits the command across multiple lines.
// The line continuation character is caret (^).
func WithWindowsMultiLine() Option {
	return func(c *Command) {
		c.cfg.style.useMultiLine = true
		c.cfg.style.lineContinuation = lineContinuationWindows
	}
}

// WithPowerShellMultiLine splits the command across multiple lines.
// The line continuation character is backtick (`).
func WithPowerShellMultiLine() Option {
	return func(c *Command) {
		c.cfg.style.useMultiLine = true
		c.cfg.style.lineContinuation = lineContinuationPowerShell
	}
}

// WithDoubleQuotes enables escaping using double quotes (").
// The default is single quotes (').
func WithDoubleQuotes() Option {
	return func(c *Command) {
		c.cfg.style.useDoubleQuotes = true
	}
}

// WithRequestTimeout enables the option -m, --max-time.
// It sets the number of seconds the request should wait
// for a response before timing out.
// Negative value seconds will be silently ignored.
func WithRequestTimeout(seconds int) Option {
	return func(c *Command) {
		if seconds < 0 {
			seconds = 0
		}
		c.cfg.requestTimeout = seconds
	}
}

// WithMaxBodySize limits the request body size (in bytes) to read.
// This prevents OOM errors on large bodies. If the body is truncated,
// the output string will be marked with "... (truncated body)".
// A value of 0 or less means a default limit (1KB).
func WithMaxBodySize(bytes int) Option {
	return func(c *Command) {
		if bytes <= 0 {
			bytes = defaultMaxBodySize
		}
		c.cfg.maxBodySize = bytes
	}
}

// WithTrustProxy enables the evaluation of "X-Forwarded-Proto" headers.
//
// This is useful when the library is used in a middleware behind a Load Balancer
// or Reverse Proxy that terminates SSL. Without this, the cURL command might
// generated as "http://" instead of "https://".
//
// SECURITY NOTE: Only use this if you trust the upstream proxy.
func WithTrustProxy() Option {
	return func(c *Command) {
		c.cfg.flags.trustProxy = true
	}
}

// WithMaskedHeaders configures a list of HTTP headers to be masked in the output.
// The values of these headers will be replaced with "*****".
// Special Case: If "Authorization" is included in the list, the password component
// of the Basic Auth flag (-u) will also be redacted.
// This option is case-insensitive.
func WithMaskedHeaders(headers ...string) Option {
	return func(c *Command) {
		for _, h := range headers {
			c.cfg.maskedHeaders[http.CanonicalHeaderKey(h)] = struct{}{}
		}
	}
}
