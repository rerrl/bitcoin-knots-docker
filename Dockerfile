FROM debian:bookworm-slim as builder

# Install dependencies for downloading and verifying
RUN apt-get update && apt-get install -y \
    wget \
    gnupg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Bitcoin Knots version
ENV BITCOIN_VERSION=29.3.knots20260508
ENV BITCOIN_URL=https://bitcoinknots.org/files/29.x/${BITCOIN_VERSION}/bitcoin-${BITCOIN_VERSION}-x86_64-linux-gnu.tar.gz
ENV SHA256SUMS_URL=https://bitcoinknots.org/files/29.x/${BITCOIN_VERSION}/SHA256SUMS
ENV SHA256SUMS_ASC_URL=https://bitcoinknots.org/files/29.x/${BITCOIN_VERSION}/SHA256SUMS.asc

# Download Bitcoin Knots and checksums
WORKDIR /tmp
RUN wget ${BITCOIN_URL} && \
    wget ${SHA256SUMS_URL} && \
    wget ${SHA256SUMS_ASC_URL}

# Import trusted builder keys from Bitcoin Knots repository
RUN wget -O - https://github.com/bitcoinknots/guix.sigs/archive/refs/heads/knots.tar.gz | tar -xz && \
    find guix.sigs-knots/builder-keys -name "*.gpg" -exec gpg --import {} \;

# Verify SHA256SUMS signature
RUN gpg --verify SHA256SUMS.asc SHA256SUMS

# Verify binary checksum
RUN grep "bitcoin-${BITCOIN_VERSION}-x86_64-linux-gnu.tar.gz" SHA256SUMS | sha256sum -c -

# Extract binaries
RUN tar -xzf bitcoin-${BITCOIN_VERSION}-x86_64-linux-gnu.tar.gz

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libatomic1 \
    libevent-2.1-7 \
    libevent-pthreads-2.1-7 \
    libminiupnpc17 \
    libnatpmp1 \
    libzmq5 \
    less \
    && rm -rf /var/lib/apt/lists/*

# Get host user ID to avoid permission issues
ARG USER_ID=1000
ARG GROUP_ID=1000

# Create bitcoin user with matching IDs
RUN groupadd -g ${GROUP_ID} bitcoin && useradd -u ${USER_ID} -g bitcoin bitcoin

# Copy binaries from builder stage
COPY --from=builder /tmp/bitcoin-*/bin/* /usr/local/bin/

# Create data directory
RUN mkdir -p /home/bitcoin/.bitcoin && \
    chown -R bitcoin:bitcoin /home/bitcoin

# Copy configuration file if it exists
COPY --chown=bitcoin:bitcoin bitcoin.conf* /tmp/
# Create entrypoint script
COPY --chown=bitcoin:bitcoin entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Switch to bitcoin user
USER bitcoin
WORKDIR /home/bitcoin

# Expose ports
# 8333: mainnet P2P
# 8332: mainnet RPC
# 18333: testnet P2P  
# 18332: testnet RPC
EXPOSE 8332 8333 18332 18333

# Default data directory
VOLUME ["/home/bitcoin/.bitcoin"]

ENTRYPOINT ["/entrypoint.sh"]
CMD ["bitcoind"]
