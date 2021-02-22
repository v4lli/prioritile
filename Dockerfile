# Build prioritile in separate container
FROM golang:buster as builder-prioritile

WORKDIR /go/src/
ENV GO111MODULE on

# Download dependencies independently for faster build
COPY prioritile/go.mod prioritile/go.sum ./
RUN go mod download

COPY prioritile ./
RUN go build
