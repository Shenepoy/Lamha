.PHONY: run build test fmt appimage

run:
	go run ./cmd/lamha

build:
	go build -o bin/lamha ./cmd/lamha

test:
	go test ./...

fmt:
	gofmt -w cmd internal

appimage:
	bash packaging/appimage/build.sh
