# Cross-compile FastPack Deploy CLI for all platforms
# Usage: make all (requires Go 1.26.4+)

BINARY=fastpack-deploy
VERSION=2.1.3
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all clean windows linux darwin

all: windows linux darwin

windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .

darwin:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .

clean:
	rm -rf dist/
