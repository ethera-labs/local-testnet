FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /src

COPY . .

RUN go build -buildvcs=false -o /out/da-server ./op-alt-da/cmd/daserver

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/da-server /usr/local/bin/da-server

ENTRYPOINT ["da-server"]
