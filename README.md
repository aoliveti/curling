# curling

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/aoliveti/curling/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aoliveti/curling)](https://pkg.go.dev/github.com/aoliveti/curling)
[![codecov](https://codecov.io/gh/aoliveti/curling/graph/badge.svg?token=3L9FOZMEJH)](https://codecov.io/gh/aoliveti/curling)
[![Go Report Card](https://goreportcard.com/badge/github.com/aoliveti/curling)](https://goreportcard.com/report/github.com/aoliveti/curling)
![GitHub License](https://img.shields.io/github/license/aoliveti/curling)

Curling is a Go library that converts [http.Request](https://pkg.go.dev/net/http#Request) objects
into [cURL](https://curl.se/) commands

## Install

```sh
go get -u github.com/aoliveti/curling
```

## Usage

The following Go code demonstrates how to create a command from an HTTP request object:

```go
req, err := http.NewRequest(http.MethodGet, "https://www.google.com", nil)
if err != nil {
    panic(err)
}
req.Header.Add("If-None-Match", "foo")

cmd, err := curling.NewFromRequest(req, curling.WithCompression())
if err != nil {
    panic(err)
}

fmt.Println(cmd)
```

```sh
curl --compressed -X 'GET' 'https://www.google.com' -H 'If-None-Match: foo'
```

### Options

```go
c, err := curling.NewFromRequest(r, opts)
```

When creating a new command, you can provide these options:

| Option                    | Description                                       |
|---------------------------|---------------------------------------------------|
| WithLongForm()            | Enables the long form for cURL options            |
| WithFollowRedirects()     | Sets the flag -L, --location                      |
| WithInsecure()            | Sets the flag -k, --insecure                      |
| WithSilent()              | Sets the flag -s, --silent                        |
| WithCompressed()          | Sets the flag --compressed                        |
| WithMultiLine()           | Generates a multiline snippet for unix-like shell |
| WithWindowsMultiLine()    | Generates a multiline snippet for Windows shell   |
| WithPowerShellMultiLine() | Generates a multiline snippet for PowerShell      |
| WithDoubleQuotes()        | Uses double quotes to escape characters           |

## License

The library is released under the MIT license. See LICENSE file.