# =================================================================
#
# Work of the U.S. Department of Defense, Defense Digital Service.
# Released as open source under the MIT License.  See LICENSE file.
#
# =================================================================

.PHONY: help
help:  ## Print the help documentation
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

#
# Go building, formatting, testing, and installing
#

fmt:  ## Format Go source code
	go fmt $$(go list ./... )

.PHONY: imports
imports: bin/goimports ## Update imports in Go source code
	bin/goimports -w \
	-local github.com/deptofdefense/icecube,github.com/deptofdefense \
	$$(find . -iname '*.go')

vet: ## Vet Go source code
	go vet github.com/deptofdefense/icecube/pkg/... # vet packages
	go vet github.com/deptofdefense/icecube/cmd/... # vet commands

tidy: ## Tidy Go source code
	go mod tidy

.PHONY: test_go
test_go: bin/errcheck bin/misspell bin/staticcheck bin/shadow ## Run Go tests
	bash scripts/test.sh

.PHONY: test_cli
test_cli: bin/icecube ## Run CLI tests
	bash scripts/test-cli.sh

install:  ## Install the CLI on current platform
	go install github.com/deptofdefense/icecube/cmd/icecube

#
# Command line Programs
#

bin/errcheck:
	go build -o bin/errcheck github.com/kisielk/errcheck

bin/goimports:
	go build -o bin/goimports golang.org/x/tools/cmd/goimports

bin/gox:
	go build -o bin/gox github.com/mitchellh/gox

bin/misspell:
	go build -o bin/misspell github.com/client9/misspell/cmd/misspell

bin/staticcheck:
	go build -o bin/staticcheck honnef.co/go/tools/cmd/staticcheck

bin/shadow:
	go build -o bin/shadow golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow

bin/icecube: ## Build icecube CLI for Darwin / amd64
	go build -o bin/icecube github.com/deptofdefense/icecube/cmd/icecube

bin/icecube_linux_amd64: bin/gox ## Build icecube CLI for Darwin / amd64
	scripts/build-release linux amd64

#
# Build Targets
#

.PHONY: build
build: bin/icecube

.PHONY: build_release
build_release: bin/gox
	scripts/build-release

.PHONY: rebuild
rebuild:
	rm -f bin/icecube
	make bin/icecube

#
# Local
#

serve_example: bin/icecube temp/ca.crt temp/server.crt   ## Serve using local binary
	bin/icecube serve \
	--addr :8080 \
	--server-cert temp/server.crt \
	--server-key temp/server.key \
	--root examples/public \
	--log temp/server.log --log-perm 644 \
	--unsafe --keylog temp/keylog

#
# Docker
#

docker_build:
	docker build -f Dockerfile --tag icecube:latest .

docker_help: ## Run the help command using docker server image
	docker run -it --rm icecube:latest help

docker_serve_example: temp/ca.crt temp/server.crt ## Serve using docker server image
	docker run -it --rm -p 8080:8080 -v $(PWD):/icecube icecube:latest serve \
	--addr :8080 \
	--server-cert /icecube/temp/server.crt \
	--server-key /icecube/temp/server.key \
	--root /icecube/examples/public \
	--unsafe --keylog /icecube/temp/keylog

docker_version:  ## Run the version command using docker server image
	docker run -it --rm icecube:latest version

#
# Certificate Targets
#

temp/ca.crt:
	mkdir -p temp
	openssl req -batch -x509 -nodes -days 365 -newkey rsa:2048 -subj "/C=US/O=Atlantis/OU=Atlantis Digital Service/CN=icecubeca" -keyout temp/ca.key -out temp/ca.crt

temp/ca.srl:
	echo '01' > temp/ca.srl

temp/index.txt:
	touch temp/index.txt

temp/index.txt.attr:
	echo 'unique_subject = yes' > temp/index.txt.attr

temp/ca.crl.pem: temp/ca.crt temp/index.txt temp/index.txt.attr
	openssl ca -batch -gencrl -config examples/conf/openssl.cnf -out temp/ca.crl.pem

temp/ca.crl.der: temp/ca.crl.pem
	openssl crl -in temp/ca.crl.pem -outform DER -out temp/ca.crl.der

temp/server.crt: temp/ca.crt temp/ca.srl temp/index.txt temp/index.txt.attr
	mkdir -p temp
	openssl genrsa -out temp/server.key 2048
	openssl req -new -config examples/conf/openssl.cnf -key temp/server.key -subj "/C=US/O=Atlantis/OU=Atlantis Digital Service/CN=icecubelocal" -out temp/server.csr
	openssl ca -batch -config examples/conf/openssl.cnf -extensions server_ext -notext -in temp/server.csr -out temp/server.crt


## Clean

.PHONY: clean
clean:  ## Clean artifacts
	rm -fr bin
