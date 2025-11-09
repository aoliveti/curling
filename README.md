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

* Converts Basic Auth to `-u` and Cookies to `-b`.
* Prioritizes `r.Host` over the `Host:` header, mimicking Go's client.
* Truncates request bodies by default (1KB) to prevent OOM errors.
* Supports multi-line output, long-form options, and quote styles.

## Install

```sh
go get -u github.com/aoliveti/curling
````

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

	req.Header = make(http.Header)
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

### Body Truncation

By default, request bodies are truncated to 1KB. The output includes a marker:
`... (truncated body, total 5000 bytes)`

You can override this limit using `WithMaxBodySize()`:

```go
// Set a 2MB limit
cmd, _ := curling.NewFromRequest(req, curling.WithMaxBodySize(2*1024*1024))
```

### Options

| Option | Description |
| --- | --- |
| `WithLongForm()` | Use long-form cURL options (e.g., `--request`) |
| `WithFollowRedirects()` | Set the flag -L, --location |
| `WithInsecure()` | Set the flag -k, --insecure |
| `WithSilent()` | Set the flag -s, --silent |
| `WithCompressed()` | Set the flag --compressed |
| `WithMultiLine()` | Use multi-line output (Unix) |
| `WithWindowsMultiLine()` | Use multi-line output (Windows CMD) |
| `WithPowerShellMultiLine()`| Use multi-line output (PowerShell) |
| `WithDoubleQuotes()` | Use double quotes for escaping |
| `WithRequestTimeout(seconds int)` | Set the flag -m, --max-time |
| `WithMaxBodySize(bytes int64)` | Override the default 1KB body read limit |

## License

The library is released under the MIT license. See [LICENSE](https://www.google.com/search?q=LICENSE) file.