# icecube

## Description

**icecube** is a HTTPS web server built in [Go](https://golang.org/). icecube uses the [net/http package](http://godoc.org/pkg/net/http) and [crypto/tls](https://godoc.org/crypto/tls#Config) packages in the Go standard library to secure communication.

## Usage

The `icecube` program has 4 sub commands: `defaults`, `help`, `serve`, and `version`.  Use `icecube serve` to launch the server.  Use `icecube defaults [tls-cipher-suites|tls-curve-preferences]` to show default configuration.  Use `icecube version` to show the current version.

Below is the usage for the `icecube serve` command.

```text
start the icecube server

Usage:
  icecube serve [flags]

Flags:
  -a, --addr string                    address that icecube will listen on (default ":8080")
      --behavior-not-found string      default behavior when a file is not found.  One of: redirect,none (default "none")
      --dry-run                        exit after checking configuration
  -h, --help                           help for serve
      --keylog string                  path to the key log output.  Also requires unsafe flag.
  -l, --log string                     path to the log output.  Defaults to stdout. (default "-")
      --public-location string         the public location of the server used for redirects
      --redirect string                address that icecube will listen to and redirect requests to the public location
  -r, --root string                    path to the document root served
      --server-cert string             path to server public cert
      --server-key string              path to server private key
      --timeout-idle string            maximum amount of time to wait for the next request when keep-alives are enabled (default "5m")
      --timeout-read string            maximum duration for reading the entire request (default "15m")
      --timeout-write string           maximum duration before timing out writes of the response (default "5m")
      --tls-cipher-suites string       list of supported cipher suites for TLS versions up to 1.2 (TLS 1.3 is not configurable)
      --tls-curve-preferences string   curve preferences (default "X25519,CurveP256,CurveP384,CurveP521")
      --tls-max-version string         maximum TLS version accepted for requests (default "1.3")
      --tls-min-version string         minimum TLS version accepted for requests (default "1.0")
      --unsafe                         allow unsafe configuration
```

### Network Encryption

**icecube** requires the use of a server certificate.  The server certificate is loaded from a PEM-encoded x509 key pair using the [LoadX509KeyPair](https://golang.org/pkg/crypto/tls/#LoadX509KeyPair) function.  The location of the key pair is specified using the `--server-cert` and `--server-key` command line flags.

## Examples

Below are the example commands and files needed to run a server.

```shell
icecube serve \
--server-cert temp/server.crt \
--server-key temp/server.key \
--root examples/public \
--behavior-not-found redirect
```

## Building

**icecube** is written in pure Go, so the only dependency needed to compile the server is [Go](https://golang.org/).  Go can be downloaded from <https://golang.org/dl/>.

This project uses [direnv](https://direnv.net/) to manage environment variables and automatically adding the `bin` and `scripts` folder to the path.  Install direnv and hook it into your shell.  The use of `direnv` is optional as you can always call icecube directly with `bin/icecube`.

If using `macOS`, follow the `macOS` instructions below.

To build a binary for development use `make bin/icecube`.  For a release call `make build_release` or call the `build-release` script directly.  Alternatively, you can always call [gox](https://github.com/mitchellh/gox) or `go build` directly.

### macOS

You can install `go` on macOS using homebrew with `brew install go`.

To install `direnv` on `macOS` use `brew install direnv`.  If using bash, then add `eval \"$(direnv hook bash)\"` to the `~/.bash_profile` file .  If using zsh, then add `eval \"$(direnv hook zsh)\"` to the `~/.zshrc` file.

## Testing

**CLI**

To run CLI testes use `make test_cli`, which uses [shUnit2](https://github.com/kward/shunit2).  If you recive a `shunit2:FATAL Please declare TMPDIR with path on partition with exec permission.` error, you can modify the `TMPDIR` environment variable in line or with `export TMPDIR=<YOUR TEMP DIRECTORY HERE>`. For example:

```shell
TMPDIR="/usr/local/tmp" make test_cli
```

**Go**

To run Go tests use `make test_go` (or `bash scripts/test.sh`), which runs unit tests, `go vet`, `go vet with shadow`, [errcheck](https://github.com/kisielk/errcheck), [staticcheck](https://staticcheck.io/), and [misspell](https://github.com/client9/misspell).

## Contributing

We'd love to have your contributions!  Please see [CONTRIBUTING.md](CONTRIBUTING.md) for more info.

## Security

Please see [SECURITY.md](SECURITY.md) for more info.

## License

This project constitutes a work of the United States Government and is not subject to domestic copyright protection under 17 USC § 105.  However, because the project utilizes code licensed from contributors and other third parties, it therefore is licensed under the MIT License.  See LICENSE file for more information.
