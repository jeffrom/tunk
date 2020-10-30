FROM golang:1.15.3 as builder

RUN mkdir /build
WORKDIR /build

COPY go.mod /build/
COPY go.sum /build/
RUN go mod download

COPY . /build

ARG VERSION=next
ARG COMMIT=none
RUN CGO_ENABLED=0 go build -o tunk.bin -ldflags "-s -w -X github.com/jeffrom/tunk/release.Version=${VERSION} -X github.com/jeffrom/tunk/release.Commit=${COMMIT}" ./cmd/tunk

FROM alpine

COPY --from=builder /build/tunk.bin /usr/local/bin/tunk

ENTRYPOINT ["tunk"]
