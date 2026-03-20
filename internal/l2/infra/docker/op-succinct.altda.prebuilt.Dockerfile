# Runtime-only image for a host-built AltDA validity proposer binary.
FROM rust:1.91-slim

WORKDIR /app

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

# Install SP1 toolchain; circuits are cached at /root/.sp1/circuits via a Docker volume.
RUN curl -L https://sp1.succinct.xyz | bash && \
    ~/.sp1/bin/sp1up -v 6.0.2 && \
    ~/.sp1/bin/cargo-prove prove --version

COPY .localnet-prebuilt/validity-proposer /usr/local/bin/validity-proposer

ENV JEMALLOC_SYS_WITH_MALLOC_CONF="background_thread:true,narenas:1,tcache:false,dirty_decay_ms:0,muzzy_decay_ms:0,abort_conf:true"

CMD ["/usr/local/bin/validity-proposer"]
