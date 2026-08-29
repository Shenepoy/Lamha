VERSION := $(shell cat VERSION)
GOLDFLAGS := -X github.com/lamha-app/lamha/internal/version.Version=$(VERSION)

.PHONY: run build test fmt appimage

run:
	go run -ldflags "$(GOLDFLAGS)" ./cmd/lamha

build:
	go build -ldflags "$(GOLDFLAGS)" -o bin/lamha ./cmd/lamha

test:
	go test ./...

fmt:
	gofmt -w cmd internal

appimage:
	LAMHA_VERSION=$(VERSION) bash packaging/appimage/build.sh
