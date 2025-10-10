KEY_SALT:=$(shell openssl rand -hex 64)
VERSION=0.0.1
GOBIN=go

# Build

.PHONY: build
build: build-linux build-windows build-darwin

.PHONY: build-linux
build-linux:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ${GOBIN} build -trimpath -ldflags="-buildid= -extldflags=-static -w -s -X dominicbreuker/goncat/pkg/config.KeySalt=${KEY_SALT} -X dominicbreuker/goncat/cmd/version.Version=${VERSION}" -o dist/goncat.elf cmd/main.go

.PHONY: build-windows
build-windows:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 ${GOBIN} build -trimpath -ldflags="-buildid= -extldflags=-static -w -s -X dominicbreuker/goncat/pkg/config.KeySalt=${KEY_SALT} -X dominicbreuker/goncat/cmd/version.Version=${VERSION}" -o dist/goncat.exe cmd/main.go

.PHONY: build-darwin
build-darwin:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 ${GOBIN} build -trimpath -ldflags="-buildid= -extldflags=-static -w -s -X dominicbreuker/goncat/pkg/config.KeySalt=${KEY_SALT} -X dominicbreuker/goncat/cmd/version.Version=${VERSION}" -o dist/goncat.macho cmd/main.go

# Linting

.PHONY: lint
lint: fmt vet staticcheck

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: staticcheck
staticcheck:
	@which staticcheck > /dev/null || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	@PATH="$(shell go env GOPATH)/bin:$$PATH" staticcheck ./...

# Test

.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit: 
	go test -cover ./...

.PHONY: test-integration
test-integration: build-linux
	@echo ""
	@echo "### ########################### ###"
	@echo "### Testing bind shell features ###"
	@echo "### ########################### ###"
	@echo ""
	TRANSPORT='tcp' TEST_SET='master-connect' docker compose -f tests/docker-compose.slave-listen.yml up --exit-code-from client
	@echo ""
	TRANSPORT='ws' TEST_SET='master-connect' docker compose -f tests/docker-compose.slave-listen.yml up --exit-code-from client
	@echo ""
	TRANSPORT='wss' TEST_SET='master-connect' docker compose -f tests/docker-compose.slave-listen.yml up --exit-code-from client
	@echo ""
	@echo "### ############################## ###"
	@echo "### Testing reverse shell features ###"
	@echo "### ############################## ###"
	@echo ""
	TRANSPORT='tcp' TEST_SET='master-listen' docker compose -f tests/docker-compose.slave-connect.yml up --exit-code-from server
	@echo ""
	TRANSPORT='ws' TEST_SET='master-listen' docker compose -f tests/docker-compose.slave-connect.yml up --exit-code-from server
	@echo ""
	TRANSPORT='wss' TEST_SET='master-listen' docker compose -f tests/docker-compose.slave-connect.yml up --exit-code-from server
