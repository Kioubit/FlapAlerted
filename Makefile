BINARY := FlapAlerted
MODULES ?=
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=${VERSION} -s -w"
BUILDFLAGS := -trimpath

.PHONY: build release release-docker release-all clean

build:
	go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY} .

release:
	CGO_ENABLED=0 go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY} .

release-docker:
	CGO_ENABLED=0 go build -tags=disable_mod_script ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY} .

release-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY}_${VERSION}_linux_amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY}_${VERSION}_linux_arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY}_${VERSION}_linux_arm

clean:
	if [ -d "bin/" ]; then find bin/ -type f -delete ;fi
	if [ -d "bin/" ]; then rm -d bin/ ;fi
