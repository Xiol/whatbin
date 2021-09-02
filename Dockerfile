FROM golang:1.17 AS builder
RUN apt update && apt install -qqy upx make
WORKDIR /usr/src/whatbin
ENV CGO_ENABLED=0
ENV GOOS=linux
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .
RUN make

FROM alpine:3
RUN apk update && apk add --no-cache ca-certificates curl chromium
RUN addgroup -S whatbin && \
    adduser -h /whatbin -S -D -H -G whatbin whatbin && \
    mkdir /whatbin
USER whatbin:whatbin
WORKDIR /whatbin
COPY --from=builder /usr/src/whatbin/cmd/whatbin/whatbin /whatbin/whatbin
COPY docker/whatbin.yml /whatbin/whatbin.yml
ENTRYPOINT ["/whatbin/whatbin"]