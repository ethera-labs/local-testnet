FROM --platform=$BUILDPLATFORM golang:1.25@sha256:5d73b7b83dd6e0258ff62832c93b6ea208fbb7727985d265fb49f75f81fc3d1f AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN go build -o localnet ./cmd/localnet

FROM ubuntu:24.04@sha256:66460d557b25769b102175144d538d88219c077c678a49af4afca6fbfc1b5252

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    git \
    bash \
    docker.io \
    docker-buildx \
    jq \
    just \
    && rm -rf /var/lib/apt/lists/*

# Install Docker Compose plugin from Docker's official repository for latest version
# Note: Older versions (e.g., v2.33.0 from Ubuntu's docker-compose-v2 package) validate
# the entire compose file even when restarting specific services, causing spurious errors
# like "env file not found" for services that aren't being restarted.
RUN curl -fsSL https://github.com/docker/compose/releases/download/v2.40.3/docker-compose-linux-$(uname -m) -o /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose && \
    ln -s /usr/local/bin/docker-compose /usr/libexec/docker/cli-plugins/docker-compose

COPY --from=ghcr.io/foundry-rs/foundry:stable@sha256:47b2c0a5759837e6b9783310113c613675f0d5f3b45885e834d6c4fca03207bc \
    /usr/local/bin/forge \
    /usr/local/bin/cast \
    /usr/local/bin/anvil \
    /usr/local/bin/chisel \
    /usr/local/bin/

COPY --from=builder /build/localnet /usr/local/bin/localnet

WORKDIR /workspace

ENTRYPOINT ["localnet"]
CMD ["--help"]

LABEL org.opencontainers.image.source="https://github.com/ethera-labs/local-testnet"
LABEL org.opencontainers.image.description="Local testnet control plane for L1 and L2 Ethereum networks"
LABEL org.opencontainers.image.licenses="Apache-2.0"
