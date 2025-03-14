KEY_SALT:=$(shell openssl rand -hex 64)
VERSION=0.0.1
GOBIN=go

.PHONY: build
build: build-linux build-windows build-darwin


.PHONY: test
test:
	go test ./...

.PHONY: build-linux
build-linux:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ${GOBIN} build -ldflags="-extldflags=-static -w -s -X dominicbreuker/goncat/pkg/config.KeySalt=${KEY_SALT} -X dominicbreuker/goncat/cmd/version.Version=${VERSION}" -o dist/goncat.elf cmd/main.go

.PHONY: build-windows
build-windows:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 ${GOBIN} build -ldflags="-extldflags=-static -w -s -X dominicbreuker/goncat/pkg/config.KeySalt=${KEY_SALT} -X dominicbreuker/goncat/cmd/version.Version=${VERSION}" -o dist/goncat.exe cmd/main.go

.PHONY: build-darwin
build-darwin:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 ${GOBIN} build -ldflags="-extldflags=-static -w -s -X dominicbreuker/goncat/pkg/config.KeySalt=${KEY_SALT} -X dominicbreuker/goncat/cmd/version.Version=${VERSION}" -o dist/goncat.macho cmd/main.go

