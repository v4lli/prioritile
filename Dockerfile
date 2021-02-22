# Build prioritile in separate container
FROM golang:buster as builder-prioritile

WORKDIR /go/src/
ENV GO111MODULE on

# Download dependencies independently for faster build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags "-linkmode external -extldflags -static" -a
