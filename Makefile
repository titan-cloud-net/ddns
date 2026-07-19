.PHONY: build build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 test vet mocks docker

build:
	go build -o ddns ./cmd

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/ddns-linux-amd64 ./cmd

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/ddns-linux-arm64 ./cmd

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/ddns-darwin-amd64 ./cmd

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/ddns-darwin-arm64 ./cmd

test:
	go test ./...

vet:
	go vet ./...

mocks:
	mockery

GOOS ?= linux

docker:
	docker build --build-arg GOOS=$(GOOS) -t ddns .
