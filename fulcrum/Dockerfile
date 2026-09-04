FROM debian:bookworm-slim as builder

# Install dependencies for downloading and verifying
RUN apt-get update && apt-get install -y \
    wget \
    gnupg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Fulcrum version (BLAKE2b PoW support)
ENV FULCRUM_VERSION=2.1.2-blake2b
# wget saves files using the URL basename, which includes the "Fulcrum-" prefix
ENV SHA256SUMS_FILE=Fulcrum-${FULCRUM_VERSION}-shasums.txt
ENV SHA256SUMS_ASC_FILE=Fulcrum-${FULCRUM_VERSION}-shasums.txt.asc
ENV FULCRUM_URL=https://github.com/privkeyio/Fulcrum/releases/download/v${FULCRUM_VERSION}/Fulcrum-${FULCRUM_VERSION}-x86_64-linux.tar.gz
ENV SHA256SUMS_URL=https://github.com/privkeyio/Fulcrum/releases/download/v${FULCRUM_VERSION}/${SHA256SUMS_FILE}
ENV SHA256SUMS_ASC_URL=https://github.com/privkeyio/Fulcrum/releases/download/v${FULCRUM_VERSION}/${SHA256SUMS_ASC_FILE}

# Download Fulcrum and checksums
WORKDIR /tmp
RUN wget ${FULCRUM_URL} && \
    wget ${SHA256SUMS_URL} && \
    wget ${SHA256SUMS_ASC_URL}

# Import the trusted Fulcrum release signer key
# Kyle Santiago <kyle@privkey.io> - key A47D99B6DB0D715D40C59A2023AE8A8EA7E24E38
RUN gpg --batch --keyserver keyserver.ubuntu.com --recv-keys A47D99B6DB0D715D40C59A2023AE8A8EA7E24E38 && \
    gpg --batch --list-keys --with-colons A47D99B6DB0D715D40C59A2023AE8A8EA7E24E38 | grep '^fpr:' | grep -q 'A47D99B6DB0D715D40C59A2023AE8A8EA7E24E38'

# Verify SHASUMS signature
RUN gpg --verify ${SHA256SUMS_ASC_FILE} ${SHA256SUMS_FILE}

# Verify binary checksum against the signed SHASUMS file
RUN grep "Fulcrum-${FULCRUM_VERSION}-x86_64-linux.tar.gz" ${SHA256SUMS_FILE} | sha256sum -c -

# Extract binaries
RUN tar -xzf Fulcrum-${FULCRUM_VERSION}-x86_64-linux.tar.gz

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies (Fulcrum statically links Qt/libstdc++/OpenSSL,
# only needs libbz2 and zlib at runtime)
RUN apt-get update && apt-get install -y \
    libbz2-1.0 \
    zlib1g \
    && rm -rf /var/lib/apt/lists/*

# Get host user ID to avoid permission issues (must match bitcoind's user so
# Fulcrum can read bitcoind's .cookie for RPC auth)
ARG USER_ID=1000
ARG GROUP_ID=1000

# Create fulcrum user with matching IDs
RUN groupadd -g ${GROUP_ID} fulcrum && useradd -u ${USER_ID} -g fulcrum fulcrum

# Copy verified binaries from builder stage
COPY --from=builder /tmp/Fulcrum-*/Fulcrum /usr/local/bin/
COPY --from=builder /tmp/Fulcrum-*/FulcrumAdmin /usr/local/bin/

# Create data directory and place config
RUN mkdir -p /home/fulcrum/db /home/fulcrum/.fulcrum && \
    chown -R fulcrum:fulcrum /home/fulcrum

# Copy configuration and entrypoint
COPY --chown=fulcrum:fulcrum fulcrum/fulcrum.conf /home/fulcrum/.fulcrum/fulcrum.conf
COPY --chown=fulcrum:fulcrum fulcrum/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Switch to fulcrum user
USER fulcrum
WORKDIR /home/fulcrum

# Expose ports
# 50001: Electrum TCP
EXPOSE 50001

# Default data directory
VOLUME ["/home/fulcrum/db"]

ENTRYPOINT ["/entrypoint.sh"]
CMD ["Fulcrum", "/home/fulcrum/.fulcrum/fulcrum.conf"]