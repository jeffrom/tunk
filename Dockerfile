FROM golang:1.17.0 as builder

RUN mkdir /build
WORKDIR /build

COPY go.mod /build/
COPY go.sum /build/
RUN go mod download

COPY . /build

ARG VERSION=next
ARG COMMIT=none
RUN CGO_ENABLED=0 go build -o tunk.bin -ldflags "-s -w -X main.Version=${VERSION} -X main.ShareDir=/usr/local/share/tunk" ./cmd/tunk

FROM alpine

RUN set -x; apk update && apk add --no-cache git

COPY --from=builder /build/tunk.bin /usr/local/bin/tunk

ENTRYPOINT ["tunk"]
