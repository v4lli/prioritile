FROM golang:alpine as gobuild

RUN apk add --no-cache git gcc libc-dev libstdc++
WORKDIR /go/src/
ENV GO111MODULE on
RUN git clone https://github.com/v4lli/prioritile.git
RUN cd prioritile && go build -ldflags "-linkmode external -extldflags -static" -a

FROM osgeo/gdal:ubuntu-full-latest

WORKDIR /app

COPY --from=gobuild /go/src/prioritile/prioritile /app/prioritile
