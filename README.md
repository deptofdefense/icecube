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

Examples:
serve --addr :8080 --server-cert server.crt --server-key server.key --root /www
serve --addr :8080 --server-key-pairs '[["server.crt", "server.key"]]' --file-systems ["/www"] --sites '{"localhost": "/www"}'  

Flags:
  -a, --addr string                    address that icecube will listen on (default ":8080")
      --aws-access-key-id string       AWS Access Key ID
      --aws-default-region string      AWS Default Region
      --aws-profile string             AWS Profile
      --aws-region string              AWS Region (overrides default region)
      --aws-secret-access-key string   AWS Secret Access Key
      --aws-session-token string       AWS Session Token
      --behavior-not-found string      default behavior when a file is not found.  One of: redirect,none (default "none")       
      --directory-index string         index file for directories (default "index.html")
      --directory-trailing-slash       append trailing slash to directories
      --dry-run                        exit after checking configuration
      --file-systems string            additional file systems in the format of a json array of strings
  -h, --help                           help for serve
      --keylog string                  path to the key log output.  Also requires unsafe flag.
  -l, --log string                     path to the log output.  Defaults to stdout. (default "-")
      --public-location string         the public location of the server used for redirects
      --redirect string                address that icecube will listen to and redirect requests to the public location
  -r, --root string                    path to the default document root served
      --server-cert string             path to default server public cert
      --server-key string              path to default server private key
      --server-key-pairs string        additional server key pairs in the format of a json array of arrays [[path to server public cert, path to server private key],...]
      --sites string                   sites hosted by the server in the format of a json map of server name to file system     
      --timeout-idle string            maximum amount of time to wait for the next request when keep-alives are enabled (default "5m")
      --timeout-read string            maximum duration for reading the entire request (default "15m")
      --timeout-write string           maximum duration before timing out writes of the response (default "5m")
      --tls-cipher-suites string       list of supported cipher suites for TLS versions up to 1.2 (TLS 1.3 is not configurable) 
      --tls-curve-preferences string   curve preferences (default "X25519,CurveP256,CurveP384,CurveP521")
      --tls-max-version string         maximum TLS version accepted for requests (default "1.3")
      --tls-min-version string         minimum TLS version accepted for requests (default "1.0")
      --unsafe                         allow unsafe configurationq
```

### Network Encryption

**icecube** requires the use of a server certificate.  The server certificate is loaded from a PEM-encoded x509 key pair using the [LoadX509KeyPair](https://golang.org/pkg/crypto/tls/#LoadX509KeyPair) function.  The location of the key pair is specified using the `--server-cert` and `--server-key` command line flags.  Alternatively, if you wish to use [Server Name Indication](https://https.cio.gov/sni/), use the `server-key-pairs` command line flag to specify multiple certificates.

## Examples

Below are the example commands and files needed to run a server.

```shell
icecube serve \
--server-cert temp/server.crt \
--server-key temp/server.key \
--root examples/public \
--behavior-not-found redirect
```

If you wish to serve multiple sites using [Server Name Indication](https://https.cio.gov/sni/), use the `server-key-pairs`, `file-systems`, and `sites` command line arguments.

```shell
icecube serve \
--server-key-pairs '[["temp/a.crt", "temp/a.key"], ["temp/b.crt", "temp/b.key"]]' \
--file-systems '["/www/a", "/www/b"]' \
--sites '{"a.localhost": "/www/a", "b.localhost" : "/www/b"}'
```

## Building

**icecube** is written in pure Go, so the only dependency needed to compile the server is [Go](https://golang.org/).  Go can be downloaded from <https://golang.org/dl/>.

This project uses [direnv](https://direnv.net/) to manage environment variables and automatically adding the `bin` and `scripts` folder to the path.  Install direnv and hook it into your shell.  The use of `direnv` is optional as you can always call icecube directly with `bin/icecube`.

If using `macOS`, follow the `macOS` instructions below.

If using `Windows`, following the `Windows` instructions below.

To build a binary for development use `make bin/icecube`.  For a release call `make build_release` or call the `build-release` script directly.  Alternatively, you can always call [gox](https://github.com/mitchellh/gox) or `go build` directly.

### macOS

You can install `go` on macOS using homebrew with `brew install go`.

To install `direnv` on `macOS` use `brew install direnv`.  If using bash, then add `eval \"$(direnv hook bash)\"` to the `~/.bash_profile` file .  If using zsh, then add `eval \"$(direnv hook zsh)\"` to the `~/.zshrc` file.

### Windows

Download the latest Windows release for `go` from [https://go.dev/dl/](https://go.dev/dl/) and install it.

For a `PowerShell` terminal, call the `.\env.ps1` file to update the local environment variables.

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
