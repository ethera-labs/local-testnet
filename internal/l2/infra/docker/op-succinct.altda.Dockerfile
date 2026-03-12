# Build stage
FROM rust:1.91 AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    libclang-dev \
    pkg-config \
    libssl-dev \
    git \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Install Go 1.24 for sp1-recursion-gnark-ffi; handle both amd64 and arm64 builders.
RUN arch="$(dpkg --print-architecture)" && \
    case "$arch" in \
      amd64) go_arch=amd64 ;; \
      arm64) go_arch=arm64 ;; \
      *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac && \
    curl -LO "https://go.dev/dl/go1.24.0.linux-${go_arch}.tar.gz" && \
    tar -C /usr/local -xzf "go1.24.0.linux-${go_arch}.tar.gz" && \
    rm "go1.24.0.linux-${go_arch}.tar.gz"
ENV PATH="/usr/local/go/bin:${PATH}"

# Cargo in official rust images uses /usr/local/cargo; cache this path explicitly.
ENV CARGO_HOME=/usr/local/cargo
ENV CARGO_NET_GIT_FETCH_WITH_CLI=true
ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
ENV RUSTUP_HOME=/usr/local/rustup
ENV RUSTUP_PROFILE=minimal

# Copy only what's needed for the build
COPY . .

# Build the validity proposer with AltDA support.
RUN --mount=type=ssh \
    --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=/usr/local/rustup \
    --mount=type=cache,target=/build/target \
    rustup set profile minimal && \
    rustup toolchain install nightly-2025-09-15 --profile minimal --component llvm-tools,rustc-dev --no-self-update && \
    cargo +nightly-2025-09-15 build --bin validity --release --features altda && \
    cp target/release/validity /build/validity-proposer

# Final stage
FROM rust:1.91-slim

WORKDIR /app

# Install required runtime dependencies
RUN apt-get update && apt-get install -y \
    curl \
    clang \
    pkg-config \
    libssl-dev \
    ca-certificates \
    git \
    libclang-dev \
    jq \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Install SP1
RUN curl -L https://sp1.succinct.xyz | bash && \
    ~/.sp1/bin/sp1up && \
    ~/.sp1/bin/cargo-prove prove --version

# Copy only the built binaries from builder
COPY --from=builder /build/validity-proposer /usr/local/bin/validity-proposer

# Run the server from its permanent location
CMD ["/usr/local/bin/validity-proposer"]
