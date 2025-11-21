<img src="assets/images/logo.png" alt="curling logo" width="256">

# curling

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aoliveti/curling)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/aoliveti/curling/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aoliveti/curling)](https://pkg.go.dev/github.com/aoliveti/curling)
[![codecov](https://codecov.io/gh/aoliveti/curling/graph/badge.svg?token=3L9FOZMEJH)](https://codecov.io/gh/aoliveti/curling)
[![Go Report Card](https://goreportcard.com/badge/github.com/aoliveti/curling)](https://goreportcard.com/report/github.com/aoliveti/curling)
![GitHub License](https://img.shields.io/github/license/aoliveti/curling)

`curling` is a Go library that converts `*http.Request` objects into cURL command strings for debugging.

## Features

* **Smart URL Handling:** Rebuilds absolute URLs for server-side requests (handling multi-hop proxies) and auto-disables globbing (`-g`) for IPv6 or array parameters.
* **RFC Compliance:** Handles multi-value headers as separate flags and strictly prioritizes `r.Host` (RFC 7230).
* **Safety:** Automatically removes `Content-Length` to prevent cURL errors and truncates request bodies (1KB default) to avoid OOM.
* **Header Privacy:** Supports masking specific headers (`*****`) or substituting them with environment variables (`$VAR`).
* **Body Security:** Supports masking the entire request body (`[CONTENT MASKED]`) to prevent PII leakage.
* **Deterministic Output:** Sorts headers and cookies alphabetically for consistent testing/logging.
* **Formatting:** Converts Basic Auth to `-u`, Cookies to `-b`, and supports multi-line output with shell-specific escaping (Bash, PowerShell, CMD).

## Install

```sh
go get -u github.com/aoliveti/curling
```

## Usage

Generate a command from an `*http.Request`. Options can be passed to `NewFromRequest`.

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aoliveti/curling"
)

func main() {
	body := `{"hello": "world"}`
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/test", strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	
	req.SetBasicAuth("user", "pass")
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc12345"})
	req.Header.Set("X-Request-ID", "12345")

	cmd, err := curling.NewFromRequest(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cmd)
}
```

**Output:**

```sh
curl -u 'user:pass' -b 'session=abc12345' --data-raw '{"hello": "world"}' 'https://api.example.com/test' -H 'X-Request-Id: 12345'
```

### Body truncation

By default, request bodies are truncated to 1KB. The output includes a marker:
`... (truncated body, total 5000 bytes)`

You can override this limit using `WithMaxBodySize()`:

```go
// Set a 2MB limit
cmd, _ := curling.NewFromRequest(req, curling.WithMaxBodySize(2*1024*1024))
```

### Server-side URL reconstruction

When used in middleware (server-side), `http.Request` often lacks the `Scheme` and `Host`. The library reconstructs the absolute URL using the following logic:

* **Scheme:** Defaults to `http`.
    * If `WithTrustProxy()` is enabled, `X-Forwarded-Proto` takes precedence (supports multi-hop).
    * Otherwise, it detects `https` if the connection state (`r.TLS`) is secure.
* **Host:** Prioritizes `r.Host` (the `Host` header), falling back to `r.URL.Host`.

### Environment variable substitution

Replace sensitive or dynamic **header values** with shell environment variables (e.g., `$TOKEN`).
This keeps debug logs clean and safe to share (no secrets leaked), while allowing the command to remain executable in terminals where the variable is set.

```go
// Use $API_TOKEN instead of the actual value in the output
cmd, _ := curling.NewFromRequest(req, curling.WithEnvVar("Authorization", "$API_TOKEN"))
// Output: curl ... -H 'Authorization: $API_TOKEN'
```

### Options

| Option                                 | Description                                                                                                 |
|----------------------------------------|-------------------------------------------------------------------------------------------------------------|
| `WithLongForm()`                       | Use long-form cURL options (e.g., `--request`)                                                              |
| `WithFollowRedirects()`                | Set the flag -L, --location                                                                                 |
| `WithInsecure()`                       | Set the flag -k, --insecure                                                                                 |
| `WithTrustProxy()`                     | Trust `X-Forwarded-Proto` for URL scheme reconstruction                                                     |
| `WithSilent()`                         | Set the flag -s, --silent                                                                                   |
| `WithCompression()`                    | Set the flag --compressed                                                                                   |
| `WithMultiLine()`                      | Use multi-line output (Unix/Bash)                                                                           |
| `WithWindowsMultiLine()`               | Use multi-line output (Windows CMD)                                                                         |
| `WithPowerShellMultiLine()`            | Use multi-line output (PowerShell)                                                                          |
| `WithDoubleQuotes()`                   | Use double quotes for escaping (Bash only)                                                                  |
| `WithRequestTimeout(seconds int)`      | Set the flag -m, --max-time                                                                                 |
| `WithMaskedHeaders(headers ...string)` | Mask specific **headers** (`*****`). Handles `-u` password redaction if `Authorization` is masked           |
| `WithMaskedBody()`                     | Replace the **request body** with `[CONTENT MASKED]` for security                                           |
| `WithEnvVar(header, variable string)`  | Replace a **header value** with an environment variable placeholder (e.g., `$TOKEN`). Priority over masking |
| `WithMaxBodySize(bytes int)`           | Override the default 1KB body read limit                                                                    |

## License

The library is released under the MIT license. See [LICENSE](https://www.google.com/search?q=LICENSE) file.