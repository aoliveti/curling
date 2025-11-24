<div><img src="assets/images/logo.png" alt="curling logo" width="256"></div>

# curling

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aoliveti/curling)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/aoliveti/curling/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aoliveti/curling)](https://pkg.go.dev/github.com/aoliveti/curling)
[![codecov](https://codecov.io/gh/aoliveti/curling/graph/badge.svg?token=3L9FOZMEJH)](https://codecov.io/gh/aoliveti/curling)
[![Go Report Card](https://goreportcard.com/badge/github.com/aoliveti/curling)](https://goreportcard.com/report/github.com/aoliveti/curling)
![GitHub License](https://img.shields.io/github/license/aoliveti/curling)

`curling` converts Go `*http.Request` objects into safe, reproducible, and executable `curl` commands.

It is designed for debugging, logging, and auditing HTTP requests in middleware or testing environments.

## Why curling?

* **URL reconstruction:** Rebuilds absolute URLs when `Scheme` and `Host` are empty (typical in server-side middleware), optionally supporting `X-Forwarded-Proto`.
* **Non-destructive reading:** Uses `http.Request.GetBody` when available (client-side) to read the payload without the overhead of buffering and restoring `req.Body`.
* **Form parsing:** Maps `application/x-www-form-urlencoded` bodies to `-d` flags and `multipart/form-data` to `-F` flags instead of dumping raw strings.
* **Content sanitization:** Truncates large bodies, omits binary/compressed data, and masks configured headers or JSON fields.
* **Deterministic output:** Sorts headers, cookies, and form keys alphabetically to ensure consistent output for logs and snapshot testing.
* **Shell compatibility:** Generates syntax-correct commands for Bash, PowerShell, and Windows CMD, applying shell-specific escaping rules.

## Table of Contents

* [Quick start](#quick-start)
* [Features](#features)
* [Examples](#examples)
    * [1. Basic POST with auth & cookies](#1-basic-post-with-auth--cookies)
    * [2. Multi-line output for logging](#2-multi-line-output-for-logging)
    * [3. Masking JSON keys](#3-masking-json-keys)
    * [4. Env vars substitution](#4-env-vars-substitution)
    * [5. Middleware: Server-side URL reconstruction](#5-middleware-server-side-url-reconstruction)
    * [6. Form data parsing](#6-form-data-parsing)
    * [7. Multipart uploads](#7-multipart-uploads)
    * [8. Binary and compressed content](#8-binary-and-compressed-content)
* [Options](#options)
    * [General configuration](#general-configuration)
    * [Shell & formatting](#shell--formatting)
    * [Privacy & security](#privacy--security)
* [License](#license)

---

## Quick start

```bash
go get -u github.com/aoliveti/curling
```

### Example

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/aoliveti/curling"
)

func main() {
	req, _ := http.NewRequest("GET", "https://api.example.com/status", nil)
	
	cmd, _ := curling.NewFromRequest(req)
	fmt.Println(cmd)
}

// Output: curl 'https://api.example.com/status'
```

## Features

* **URL reconstruction:** Rebuilds absolute URLs for server-side requests, prioritizing `X-Forwarded-Proto` (if enabled) and `r.Host`.
* **Data parsing:**
    * Parses `application/x-www-form-urlencoded` bodies into individual `-d` flags.
    * Parses `multipart/form-data` bodies into `-F` flags, replacing file content with placeholders.
* **Content safety:** Detects and omits binary media types (e.g., images, PDF) and compressed streams (`Content-Encoding: gzip`) to prevent terminal corruption.
* **Performance:** Utilizes `http.Request.GetBody` when available to read the body without buffering overhead.
* **RFC compliance:** Handles multi-value headers as separate flags and respects RFC 7230 Host prioritization.
* **Redaction:** Supports masking of headers, specific JSON keys, or the entire request body.
* **Shell compatibility:** Generates syntax for Bash, PowerShell, and Windows CMD.

## Examples

### 1. Basic POST with auth & cookies

```go
req, _ := http.NewRequest("POST", "https://api.example.com/test", strings.NewReader(`{"hello":"world"}`))
req.SetBasicAuth("user", "pass")
req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

cmd, _ := curling.NewFromRequest(req)

// Output:
// curl -u 'user:pass' -b 'session=abc' --data-raw '{"hello":"world"}' 'https://api.example.com/test'
```

### 2. Multi-line output for logging

Generate readable commands for log files using `WithMultiLine()`.

```go
// Default (Bash style)
cmd, _ := curling.NewFromRequest(req, curling.WithMultiLine())

// PowerShell style
cmd, _ := curling.NewFromRequest(req, curling.WithMultiLine(), curling.WithTargetShell(curling.PowerShell))
```

### 3. Masking JSON keys

Granularly redact sensitive fields inside a JSON body while keeping the structure for debugging.

```go
// Masks "password" recursively in the JSON body
cmd, _ := curling.NewFromRequest(req, curling.WithMaskedJSONFields("password"))

// Output:
// curl ... --data-raw '{"user":"admin","password":"*****"}'
```

### 4. Env vars substitution

Generate shareable commands using shell variables instead of hardcoded secrets.

```go
// Use $API_TOKEN instead of the actual value
cmd, _ := curling.NewFromRequest(req, curling.WithEnvVar("Authorization", "$API_TOKEN"))

// Output:
// curl ... -H 'Authorization: $API_TOKEN'
```

### 5. Middleware: Server-side URL reconstruction

When used in middleware, `http.Request` often lacks Scheme and Host. `curling` reconstructs the URL using the following logic:

* **Scheme**: Defaults to `http`.

    * If `WithTrustProxy()` is enabled, `X-Forwarded-Proto` takes precedence.
    * Otherwise, detects `https` via `r.TLS`.
* **Host**: Prioritizes `r.Host`, falling back to `r.URL.Host`.

### 6. Form data parsing

Parses `application/x-www-form-urlencoded` into `-d` flags.

```go
// req is a POST with Content-Type: application/x-www-form-urlencoded
// Body: "q=golang&sort=desc"
cmd, _ := curling.NewFromRequest(req)

// Output:
// curl -d 'q=golang' -d 'sort=desc' 'https://api.example.com/search'
```

### 7. Multipart uploads

Converts `multipart/form-data` payloads into `-F` flags. File content is omitted with a placeholder.

```go
// req is a multipart request containing:
// - Text field "username": "alice"
// - File field "avatar": filename "avatar.jpg"
cmd, _ := curling.NewFromRequest(req)

// Output:
// curl -F 'avatar=@avatar.jpg (OMITTED)' -F 'username=alice' 'https://api.example.com/upload'
```

### 8. Binary and compressed content

Safe logging for binary streams (image/*, application/pdf) or compressed bodies (gzip).

```go
// Scenario A: req has header "Content-Encoding: gzip"
cmd, _ := curling.NewFromRequest(req)
// Output: curl ... --data-raw '[ENCODED DATA OMITTED: Content-Encoding: gzip]' ...

// Scenario B: req has header "Content-Type: image/jpeg"
cmd, _ := curling.NewFromRequest(req)
// Output: curl ... --data-raw '[BINARY DATA OMITTED: image/jpeg]' ...
```

## Options

### General configuration

| Option                            | Description                                                                        |
|-----------------------------------|------------------------------------------------------------------------------------|
| `WithLongForm()`                  | Use long-form cURL options (e.g., `--request` instead of `-X`)                     |
| `WithSilent()`                    | Set the flag `-s`, `--silent`                                                      |
| `WithCompression()`               | Set the flag `--compressed`                                                        |
| `WithFollowRedirects()`           | Set the flag `-L`, `--location`                                                    |
| `WithInsecure()`                  | Set the flag `-k`, `--insecure`                                                    |
| `WithRequestTimeout(seconds int)` | Set the flag `-m`, `--max-time`                                                    |
| `WithMaxBodySize(bytes int)`      | Override the default 1KB limit. Truncation disables form parsing and JSON masking. |
| `WithTrustProxy()`                | Trust `X-Forwarded-Proto` for URL scheme reconstruction                            |

### Shell & formatting

| Option                   | Description                                                                                                    |
|--------------------------|----------------------------------------------------------------------------------------------------------------|
| `WithMultiLine()`        | Use multi-line output. Uses default `\` separator unless a target shell is set.                                |
| `WithTargetShell(shell)` | Set target shell syntax (`POSIX`, `PowerShell`, `WindowsCMD`). Configures escaping and line continuation char. |
| `WithDoubleQuotes()`     | Use double quotes for escaping (Bash only). Ignored for Windows/PowerShell.                                    |

#### Shell compatibility matrix

| Shell           | Escaping Strategy                   | Notes                                              |
|-----------------|-------------------------------------|----------------------------------------------------|
| **Bash / Zsh**  | Single (`'`) or Double (`"`) quotes | Default. Configurable via `WithDoubleQuotes()`.    |
| **PowerShell**  | Backticks (`` ` ``)                 | Enforces double quotes for safety.                 |
| **Windows CMD** | MS C Runtime rules                  | Enforces double quotes; robust backslash handling. |

### Privacy & security

| Option                          | Description                                                                                                 | Precedence                       |
|---------------------------------|-------------------------------------------------------------------------------------------------------------|----------------------------------|
| `WithEnvVar(header, var)`       | Replace a header value with a shell variable (e.g., `$TOKEN`).                                              | Highest (Overrides Masking)      |
| `WithMaskedHeaders(keys...)`    | Mask specific headers (`*****`). Handles `-u` password redaction if Authorization is masked.                | Normal                           |
| `WithMaskedBody()`              | Replace the request body with `[CONTENT MASKED]`. Zero-copy optimization.                                   | Highest (Overrides JSON Masking) |
| `WithMaskedJSONFields(keys...)` | Mask specific JSON keys in the body (`*****`). Falls back to total masking if JSON is invalid or truncated. | Normal                           |

## License

The library is released under the MIT license. See [LICENSE](LICENSE) file.