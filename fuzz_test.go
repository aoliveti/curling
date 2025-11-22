package curling

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Fuzz_NewFromRequest(f *testing.F) {
	// Add seed corpus entries covering various HTTP methods and edge cases.
	// The final byte parameter serves as a bitmask to toggle options deterministically.
	f.Add("GET", "https://example.com", "", "X-Key", "value", byte(0))
	f.Add("POST", "https://foo.com/q?a='b'", "{'a':1}", "Authorization", "Bearer 123", byte(255))
	f.Add("PUT", "https://'test'.com", "\"", "Host", "host.com", byte(10))
	f.Add("PATCH", "https://unicode.com", "空😎", "X-Test", "值", byte(50))
	f.Add("DELETE", "https://attack.com", "`rm -rf /`", "X-Evil", "$(uname)", byte(100))
	f.Add("POST", "https://api.local", "data", "Set-Cookie", "id=1; secure", byte(200))

	// Edge Case: Complex URL to test smart globbing (brackets and braces).
	f.Add("GET", "https://example.com/a[b]{c}?d=e", "", "X-Glob", "test", byte(90))

	// Edge Case: Large Body to test truncation logic (5KB string).
	f.Add("POST", "https://api.com", strings.Repeat("A", 5000), "X-Large", "true", byte(0))

	// Edge Case: Multi-value header simulation using pipe separator.
	f.Add("GET", "https://example.com", "", "X-List", "item1|item2|item3", byte(0))

	f.Fuzz(func(t *testing.T, method, urlStr, body, hKey, hVal string, flags byte) {
		// Parse the URL and skip the test if the input generates an invalid URL structure.
		u, err := url.Parse(urlStr)
		if err != nil || u.Scheme == "" {
			t.Skip()
		}

		// Construct the header map. We split the value string by pipe (|)
		// to simulate multi-value headers, which ensures our separate -H flag logic is exercised.
		header := http.Header{}
		if len(hKey) > 0 {
			values := strings.Split(hVal, "|")
			for _, v := range values {
				header.Add(hKey, v)
			}
		}

		r := &http.Request{
			Method: method,
			URL:    u,
			Header: header,
			Body:   io.NopCloser(strings.NewReader(body)),
		}

		// Use the flags byte to deterministically set properties on the request.
		// The first bit enables Basic Auth and the second bit adds a test cookie.
		if flags&1 != 0 {
			r.SetBasicAuth("fuzzUser", "fuzzPass")
		}
		if flags&2 != 0 {
			r.AddCookie(&http.Cookie{Name: "fuzz-cookie", Value: "yummy"})
		}

		var opts []Option

		// Determine the shell style based on the third and fourth bits.
		// This cycles through default Bash, MultiLine, Windows CMD, and PowerShell modes.
		switch (flags >> 2) & 3 {
		case 1:
			opts = append(opts, WithMultiLine())
		case 2:
			opts = append(opts, WithTargetShell(WindowsCMD))
		case 3:
			opts = append(opts, WithTargetShell(PowerShell))
		}

		// Enable Double Quotes if the fifth bit is set.
		if flags&16 != 0 {
			opts = append(opts, WithDoubleQuotes())
		}

		// Enable header masking for the current header key if the sixth bit is set.
		if flags&32 != 0 && len(hKey) > 0 {
			opts = append(opts, WithMaskedHeaders(hKey))
		}

		// Enable environment variable substitution if the seventh bit is set.
		if flags&64 != 0 && len(hKey) > 0 {
			opts = append(opts, WithEnvVar(hKey, "$ENV_VAR"))
		}

		// Enable body masking if the eighth bit is set.
		if flags&128 != 0 {
			opts = append(opts, WithMaskedBody())
		}

		// Execute the function under test.
		cmd, err := NewFromRequest(r, opts...)

		// If the function returns an error (e.g. invalid URL construction inside NewFromRequest),
		// we simply return as this is expected behavior for random inputs.
		if err != nil {
			return
		}

		// Verify that a successful execution produces a valid command string.
		if cmd == nil {
			t.Errorf("Command is nil despite no error")
			return
		}

		s := cmd.String()
		if s == "" {
			t.Errorf("Generated command is empty")
		}
		if !strings.HasPrefix(s, "curl") {
			t.Errorf("Command should start with 'curl', got: %s", s)
		}

		// Verify security properties. If body masking was requested, the original
		// body content should not appear in the output.
		if (flags&128 != 0) && len(body) > 10 && strings.Contains(s, body) {
			t.Errorf("Security failure: Body was supposed to be masked but found original content")
		}
	})
}
