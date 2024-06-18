.PHONY: build

build:
	go build -trimpath -ldflags "-s -w" -o ./cmd/whatbin ./cmd/whatbin
