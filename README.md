# Bitcoin Knots Docker Setup (v29.1.knots20250903)

This repository provides a Docker-based setup for running Bitcoin Knots and electrs

**Current Bitcoin Knots Version**: 29.1.knots20250903  
**Source**: URLs pulled from https://bitcoinknots.org/

## How to use

1. Clone this repository

2. Copy the example environment file and customize it:

```bash
cp .env.example .env
```

Edit `.env` to set your desired paths and configuration. For development, the defaults are fine.

3. Create a `bitcoin.conf` file in the repository root directory. This file will be copied into the container during the build process.

4. Create your data directories (if using default paths):

```bash
mkdir bitcoin-data electrs-data
```

5. Build and start the containers:

```bash
docker compose up --build
```

5. If you see an error from electrs saying it can't find the cookie file, make sure in your bitcoin.conf you're allowing the proper docker subnet IP. You can find this by running:

```bash
# Find the docker network name that these containers are on
docker network ls

# Find the subnet of the docker network
docker network inspect <network name> | grep Subnet

# update the bitcoin.conf file with the subnet
```

## How it works

When you run `docker-compose up --build`, the following happens:

### Bitcoin Knots Security & Authenticity

The Dockerfile implements a robust security verification process to ensure the authenticity of Bitcoin Knots binaries:

1. **Multi-stage build**: Uses a separate builder stage to download and verify binaries before copying to the runtime container
2. **Cryptographic verification**: 
   - Downloads Bitcoin Knots binaries, SHA256SUMS, and SHA256SUMS.asc files
   - Imports trusted builder keys from the official Bitcoin Knots guix.sigs repository
   - Verifies the GPG signature on SHA256SUMS using these trusted keys
   - Validates the binary checksum against the signed SHA256SUMS file
3. **Clean runtime**: Only verified binaries are copied to the final runtime container

### Container Architecture

- **bitcoind container**: Runs Bitcoin Knots with user/group ID matching your host system to avoid permission issues
- **electrs container**: Provides an Electrum server interface to the Bitcoin node
- Both containers communicate over a Docker network, with electrs connecting to bitcoind's RPC interface

### Data Storage

**Development/Testing**: By default, data is stored in local directories (`./bitcoin-data` and `./electrs-data`)

**Production**: For production deployments, you should use external storage volumes (like dedicated SSDs) mounted to your host system:

1. **Mount your external storage** (e.g., SSD) to your host system:
   ```bash
   # Example: mount external SSD to /mnt/bitcoin-storage
   sudo mount /dev/sdX1 /mnt/bitcoin-storage
   sudo mkdir -p /mnt/bitcoin-storage/bitcoin-data
   sudo mkdir -p /mnt/bitcoin-storage/electrs-data
   ```

2. **Update your .env file** to use the mounted paths:
   ```bash
   # Production paths in .env
   BITCOIN_DATA_PATH=/mnt/bitcoin-storage/bitcoin-data
   ELECTRS_DATA_PATH=/mnt/bitcoin-storage/electrs-data
   ELECTRS_SERVER_BANNER=Production Bitcoin Node
   ```

This approach provides better performance, dedicated storage space, and easier backup/migration capabilities. Make sure to set proper ownership and permissions on the mounted directories to match your container user IDs.
