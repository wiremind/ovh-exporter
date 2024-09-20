GO?=go

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

BINARY_NAME=ovh-exporter
# TODO less bruteforce ?
BINARY_FILES=$(shell find ${PWD} -type f -name "*.go")
GOFMT_FILES?=$(shell find . -not -path "./vendor/*" -type f -name '*.go')

VERSION ?= $(shell git describe --match 'v[0-9]*' --dirty='.m' --always)
REVISION=$(shell git rev-parse HEAD)$(shell if ! git diff --no-ext-diff --quiet --exit-code; then echo .m; fi)
PKG=github.com/wiremind/ovh-exporter

# Control if static or dynamically linked (static by default)
export CGO_ENABLED:=0

GO_GCFLAGS?=
GO_LDFLAGS=-ldflags '-X $(PKG)/version.Version=$(VERSION) -X $(PKG)/version.Revision=$(REVISION) -X $(PKG)/version.Package=$(PKG)'

.PHONY: build
build: ${BINARY_NAME}

${BINARY_NAME}: ${BINARY_FILES}
	${GO} build ${GO_GCFLAGS} ${GO_LDFLAGS} -o $@ cmd/${BINARY_NAME}/*.go
	strip -x $@

## Lints all the go code in the application.
.PHONY: lint
lint: dependencies
	gofmt -w $(GOFMT_FILES)
	$(GOBIN)/goimports -w $(GOFMT_FILES)
	$(GOBIN)/gofumpt -l -w $(GOFMT_FILES)
	$(GOBIN)/gci write $(GOFMT_FILES) --skip-generated
	$(GOBIN)/golangci-lint run

## Loads all the dependencies to vendor directory
.PHONY: dependencies
dependencies:
	go install golang.org/x/tools/cmd/goimports@v0.25.0
	go install mvdan.cc/gofumpt@v0.7.0
	go install github.com/daixiang0/gci@v0.13.5
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61
	go mod vendor
	go mod tidy
