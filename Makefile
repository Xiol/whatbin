.PHONY: build

build:
	go build -trimpath -ldflags "-s -w" -o ./cmd/whatbin ./cmd/whatbin

upx: build
	upx -7 cmd/whatbin/whatbin
