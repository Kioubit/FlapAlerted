# Makefile for FlapAlertedPro

BINARY=FlapAlertedPro
MODULES=mod_log,mod_httpAPI
VERSION=`git describe --tags`
LDFLAGS=-ldflags "-X main.Version=${VERSION}"
BUILDFLAGS=-trimpath

build:
	go build -tags=${MODULES} -o bin/${BINARY} .

release:
	CGO_ENABLED=0 GOOS=linux go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY}_${VERSION}_linux_amd64.bin .

release-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY}_${VERSION}_linux_amd64.bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags=${MODULES} ${BUILDFLAGS} ${LDFLAGS} -o bin/${BINARY}_${VERSION}_linux_arm64.bin

clean:
	if [ -d "bin/" ]; then find bin/ -type f -delete ;fi
	if [ -d "bin/" ]; then rm -d bin/ ;fi
