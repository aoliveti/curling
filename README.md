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

* **Middleware Ready:** Automatically reconstructs URLs for server-side requests where Scheme and Host are missing, and handles multi-hop proxies (`X-Forwarded-Proto`).
* **Secure by Default:** Prevents secrets leakage via robust masking options for headers, body, and specific JSON keys.
* **Deterministic:** Sorts headers and cookies alphabetically, ensuring stable logs and reliable snapshot testing.
* **Shell Aware:** Generates syntax-correct commands for Bash, PowerShell, and Windows CMD, handling complex escaping (e.g., nested quotes, JSON) correctly.

## Table of Contents

* [Quick Start](#quick-start)
* [Features](#features)
* [Examples](#examples)

    * [1. Basic POST with Auth & Cookies](#1-basic-post-with-auth--cookies)
    * [2. Multi-line Output for Logging](#2-multi-line-output-for-logging)
    * [3. Masking JSON Keys](#3-masking-json-keys)
    * [4. Env Vars Substitution](#4-env-vars-substitution)
    * [5. Middleware: Server-side URL Reconstruction](#5-middleware-server-side-url-reconstruction)
* [Options](#options)

    * [General Configuration](#general-configuration)
    * [Shell & Formatting](#shell--formatting)
    * [Privacy & Security](#privacy--security)
* [License](#license)

---

## Quick Start

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
	
	// Generate command with default options
	cmd, _ := curling.NewFromRequest(req)
	fmt.Println(cmd)
}

// Output: curl 'https://api.example.com/status'
```

## Features

* **Smart URL Handling:** Rebuilds absolute URLs for server-side requests and auto-disables globbing (`-g`) for IPv6 or array parameters.
* **RFC Compliance:** Handles multi-value headers as separate flags (`-H "A: 1" -H "A: 2"`) and strictly prioritizes `r.Host` (RFC 7230).
* **Safety:** Automatically removes `Content-Length` to prevent cURL errors and truncates request bodies (1KB default) to avoid OOM.
* **Privacy & Security:** Supports masking headers (`*****`), specific JSON keys, or the entire body (`[CONTENT MASKED]`). Also supports environment variable substitution (`$VAR`).
* **Formatting:** Converts Basic Auth to `-u`, Cookies to `-b`, and supports multi-line output with shell-specific escaping.

## Examples

### 1. Basic POST with Auth & Cookies

```go
req, _ := http.NewRequest("POST", "https://api.example.com/test", strings.NewReader(`{"hello":"world"}`))
req.SetBasicAuth("user", "pass")
req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

cmd, _ := curling.NewFromRequest(req)

// Output:
// curl -u 'user:pass' -b 'session=abc' --data-raw '{"hello":"world"}' 'https://api.example.com/test'
```

### 2. Multi-line Output for Logging

Generate readable commands for log files using `WithMultiLine()`.

```go
// Default (Bash style)
cmd, _ := curling.NewFromRequest(req, curling.WithMultiLine())

// PowerShell style
cmd, _ := curling.NewFromRequest(req, curling.WithMultiLine(), curling.WithTargetShell(curling.PowerShell))
```

### 3. Masking JSON Keys

Granularly redact sensitive fields inside a JSON body while keeping the structure for debugging.

```go
// Masks "password" recursively in the JSON body
cmd, _ := curling.NewFromRequest(req, curling.WithMaskedJSONFields("password"))

// Output:
// curl ... --data-raw '{"user":"admin","password":"*****"}'
```

### 4. Env Vars Substitution

Generate shareable commands using shell variables instead of hardcoded secrets.

```go
// Use $API_TOKEN instead of the actual value
cmd, _ := curling.NewFromRequest(req, curling.WithEnvVar("Authorization", "$API_TOKEN"))

// Output:
// curl ... -H 'Authorization: $API_TOKEN'
```

### 5. Middleware: Server-side URL Reconstruction

When used in middleware, `http.Request` often lacks Scheme and Host. `curling` reconstructs the URL using the following logic:

* **Scheme**: Defaults to `http`.

    * If `WithTrustProxy()` is enabled, `X-Forwarded-Proto` takes precedence.
    * Otherwise, detects `https` via `r.TLS`.
* **Host**: Prioritizes `r.Host`, falling back to `r.URL.Host`.

## Options

### General Configuration

| Option                            | Description                                                    |
|-----------------------------------|----------------------------------------------------------------|
| `WithLongForm()`                  | Use long-form cURL options (e.g., `--request` instead of `-X`) |
| `WithSilent()`                    | Set the flag `-s`, `--silent`                                  |
| `WithCompression()`               | Set the flag `--compressed`                                    |
| `WithFollowRedirects()`           | Set the flag `-L`, `--location`                                |
| `WithInsecure()`                  | Set the flag `-k`, `--insecure`                                |
| `WithRequestTimeout(seconds int)` | Set the flag `-m`, `--max-time`                                |
| `WithMaxBodySize(bytes int)`      | Override the default 1KB body read limit                       |
| `WithTrustProxy()`                | Trust `X-Forwarded-Proto` for URL scheme reconstruction        |

### Shell & Formatting

| Option                      | Description                                      |
|-----------------------------|--------------------------------------------------|
| `WithMultiLine()`           | Use multi-line output (Unix/Bash style `\`)      |
| `WithWindowsMultiLine()`    | Use multi-line output (Windows CMD style `^`)    |
| `WithPowerShellMultiLine()` | Use multi-line output (PowerShell style `` ` ``) |
| `WithDoubleQuotes()`        | Use double quotes for escaping (Bash only)       |

### Shell & Formatting

| Option                   | Description                                                                                                    |
|--------------------------|----------------------------------------------------------------------------------------------------------------|
| `WithMultiLine()`        | Use multi-line output. Uses default `\` separator unless a target shell is set.                                |
| `WithTargetShell(shell)` | Set target shell syntax (`POSIX`, `PowerShell`, `WindowsCMD`). Configures escaping and line continuation char. |
| `WithDoubleQuotes()`     | Use double quotes for escaping (Bash only). Ignored for Windows/PowerShell.                                    |

#### Shell Compatibility Matrix

| Shell           | Escaping Strategy                   | Notes                                              |
|-----------------|-------------------------------------|----------------------------------------------------|
| **Bash / Zsh**  | Single (`'`) or Double (`"`) quotes | Default. Configurable via `WithDoubleQuotes()`.    |
| **PowerShell**  | Backticks (`` ` ``)                 | Enforces double quotes for safety.                 |
| **Windows CMD** | MS C Runtime rules                  | Enforces double quotes; robust backslash handling. |

### Privacy & Security

| Option                          | Description                                                                                                 | Precedence                       |
|---------------------------------|-------------------------------------------------------------------------------------------------------------|----------------------------------|
| `WithEnvVar(header, var)`       | Replace a header value with a shell variable (e.g., `$TOKEN`).                                              | Highest (Overrides Masking)      |
| `WithMaskedHeaders(keys...)`    | Mask specific headers (`*****`). Handles `-u` password redaction if Authorization is masked.                | Normal                           |
| `WithMaskedBody()`              | Replace the request body with `[CONTENT MASKED]`. Zero-copy optimization.                                   | Highest (Overrides JSON Masking) |
| `WithMaskedJSONFields(keys...)` | Mask specific JSON keys in the body (`*****`). Falls back to total masking if JSON is invalid or truncated. | Normal                           |

## License

The library is released under the MIT license. See [LICENSE](LICENSE) file.