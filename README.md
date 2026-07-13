# Bitcoin Knots Docker Setup (v29.3.knots20260508)

This repository provides a Docker-based setup for running Bitcoin Knots and electrs, with an optional Tor hidden service for remote Electrum access.

**Current Bitcoin Knots Version**: 29.3.knots20260508
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

Optionally create a `tor-data` directory if you plan to enable Tor (see Tor section below).

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
- **tor container** (optional): Exposes electrs as a Tor hidden service so you can connect from remote wallets like Sparrow via Tor
- Containers communicate over a Docker network, with electrs connecting to bitcoind's RPC interface, and tor connecting to electrs

### Data Storage

**Development/Testing**: By default, data is stored in local directories (`./bitcoin-data`, `./electrs-data`, and `./tor-data`)

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

## Tor Hidden Service (Remote Access)

The tor container creates a Tor hidden service that exposes electrs's Electrum port (50001) as a `.onion` address. This lets you connect to your node remotely from wallets that support Tor (like Sparrow Wallet).

### How it works

1. The tor container builds from `tor/Dockerfile` (Alpine + Tor)
2. `tor/torrc` configures a hidden service mapping port 50001 to `electrs:50001`
3. On startup, `tor/entrypoint.sh` waits for Tor to generate the hostname and prints the `.onion` address to the container logs
4. The onion address and its private key persist in `./tor-data/` (the mounted volume), so the address stays the same across restarts

### Enabling Tor (disabled by default)

Tor is **disabled by default** — the tor container only starts when you activate it.

1. **Uncomment the profile line** in your `.env` file:

   ```
   COMPOSE_PROFILES=tor
   ```

   This tells Docker Compose to include the `tor` profile on every `docker compose up`.

2. **Create the Tor data directory** (stores the `.onion` private key so the address is stable):

   ```bash
   mkdir tor-data
   ```

3. **Make sure `TOR_DATA_PATH` is set** — `.env.example` already includes `TOR_DATA_PATH=./tor-data`. If you copied `.env.example` to `.env` before this change, add it manually:

   ```
   TOR_DATA_PATH=./tor-data
   ```

That's it. Next `docker compose up` will build and start the tor container automatically.

To disable it again, just comment out or remove `COMPOSE_PROFILES=tor` from `.env`.

### Getting the onion address

After starting the containers, read it from the tor container logs:

```bash
docker compose logs tor
```

You'll see output like:

```
tor  | Waiting for Tor hidden service to be ready...
tor  | ==========================================
tor  |   Tor Hidden Service: xyzabc123def456.onion
tor  |   Connect with Sparrow Wallet:
tor  |     Server:   xyzabc123def456.onion
tor  |     Port:     50001
tor  |     Protocol: TCP (SSL disabled)
tor  | ==========================================
```

You can also read the hostname file directly from the host:

```bash
cat ./tor-data/hostname
```

### Configuring Sparrow Wallet

1. Open Sparrow → Preferences → Connection
2. Set **Server Type** to "Electrum Server"
3. Set **URL** to your `.onion` address (e.g., `xyzabc123def456.onion:50001`)
4. Make sure Sparrow has Tor enabled in its settings (Tools → Restart in Tor mode, or configure the Tor SOCKS proxy under Preferences → Connection → Use Tor)
5. Click "Test Connection"

### Important notes

- **Privacy**: The `.onion` address is public by design — it's how other nodes reach your service. The private key (in `./tor-data/private_key`) is what keeps it secure. Protect the tor-data directory.
- **Back up `./tor-data/`** — losing the private key means getting a new `.onion` address and reconfiguring all connected wallets.
- **Only electrs is exposed** via the hidden service. The bitcoind RPC and P2P ports are not routed through Tor — they remain local.
- **The tor container is disabled by default** via Docker Compose profiles. It only starts if `COMPOSE_PROFILES=tor` is set in your `.env` file. To disable an existing setup, just comment out that line.
