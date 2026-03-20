# syntax=docker/dockerfile:1.7
# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev linux-headers git

WORKDIR /go-ethereum

# Some local op-geth forks vendor the registry module in-tree.
# Use it when present so container builds do not depend on fetching a private repo.
COPY . /go-ethereum
RUN if [ -f registry/go.mod ]; then \
      go mod edit -replace github.com/ethera-labs/registry=./registry; \
    fi

RUN --mount=type=cache,target=/go/pkg/mod \
    cd /go-ethereum && go mod download

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd /go-ethereum && go run build/ci.go install -static ./cmd/geth

FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-ethereum/build/bin/geth /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geth"]

ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
