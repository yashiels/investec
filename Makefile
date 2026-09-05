BINARY := investec
PKG := ./cmd/investec
VERSION := $(shell grep '^MARKETING_VERSION=' version.env | cut -d= -f2)
LDFLAGS := -X github.com/yashiels/investec/internal/cli.Version=$(VERSION)

.PHONY: build test lint vet fmt tidy clean install

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

lint: vet
	@gofmt -l . | grep . && { echo "gofmt needed"; exit 1; } || echo "gofmt clean"

tidy:
	go mod tidy

install: build
	install -m 0755 $(BINARY) $(GOPATH)/bin/$(BINARY)

clean:
	rm -f $(BINARY)
