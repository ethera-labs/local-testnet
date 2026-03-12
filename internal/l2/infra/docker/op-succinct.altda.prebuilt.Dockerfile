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

RUN curl -L https://sp1.succinct.xyz | bash && \
    ~/.sp1/bin/sp1up && \
    ~/.sp1/bin/cargo-prove prove --version

COPY .localnet-prebuilt/validity-proposer /usr/local/bin/validity-proposer

CMD ["/usr/local/bin/validity-proposer"]
