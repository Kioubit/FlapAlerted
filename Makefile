BINARY := FlapAlerted
MODULES ?=
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

TAGS := $(if $(MODULES),-tags=$(MODULES),)
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -s -w"
BUILDFLAGS := -trimpath
GO_BUILD = go build $(TAGS) $(BUILDFLAGS) $(LDFLAGS)

.PHONY: build release release-all clean

build:
	$(GO_BUILD) -o bin/$(BINARY) .

release:
	CGO_ENABLED=0 $(GO_BUILD) -o bin/$(BINARY) .

release-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD) -o bin/$(BINARY)_$(VERSION)_linux_amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO_BUILD) -o bin/$(BINARY)_$(VERSION)_linux_arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm $(GO_BUILD) -o bin/$(BINARY)_$(VERSION)_linux_arm

clean:
	rm -rf "./bin/"
