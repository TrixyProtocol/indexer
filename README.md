# Trixy Protocol - Flow Indexer

A blockchain indexer for the Trixy Protocol on Flow Network. Indexes prediction market events including market creation, bet placement, resolution, winnings claims, and yield operations.

## Features

- 🔄 Real-time event indexing from Flow blockchain
- 📊 PostgreSQL storage with GORM
- 🔍 Continuous block monitoring with automatic sync state management
- 🎯 Event tracking:
  - MarketCreated
  - BetPlaced
  - MarketResolved
  - WinningsClaimed
  - YieldDeposited
  - YieldWithdrawn
- ✅ Code quality with golangci-lint
- 🚀 Batch processing for efficient indexing

## Prerequisites

- Go 1.21+
- PostgreSQL 13+
- Flow gRPC access node endpoint

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd indexer
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Install golangci-lint (optional, for development):
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   export PATH=$PATH:$HOME/go/bin
   ```

## Configuration

1. Copy the example config:
   ```bash
   cp config-example.yaml config.yaml
   ```

2. Edit `config.yaml`:
   ```yaml
   mode: "indexer"
   dbType: "postgres"
   dbHost: "localhost"
   dbPort: 5432
   dbUser: "your_username"
   dbPass: "your_password"  # Leave empty for passwordless auth
   dbName: "trixy-flow-indexer"
   rpcEndpoint: "access.devnet.nodes.onflow.org:9000"
   network: "flow-testnet"
   networksFile: "networks.json"
   indexWorkers: 2
   forceResyncOnEveryStart: false
   migrateOnStart: true
   blockBatchSize: 50
   ```

3. Configure networks in `networks.json`:
   ```json
   {
     "flow-testnet": {
       "TrixyProtocol": {
         "address": "0xe4a8713903104ee5",
         "startBlock": 287066000
       }
     }
   }
   ```

## Database Setup

Create the PostgreSQL database:
```bash
psql -U postgres
CREATE DATABASE "trixy-flow-indexer";
```

Tables are auto-created on first run when `migrateOnStart: true`.

## Usage

### Run the indexer:
```bash
go run main.go
```

### Build binary:
```bash
go build -o trixy-indexer
./trixy-indexer
```

### Development - Run linter:
```bash
golangci-lint run --fix
```

## Architecture

### Project Structure
```
.
├── main.go                 # Entry point and main indexing loop
├── config/
│   ├── config.go          # Configuration management
│   ├── db.go              # Database connection
│   └── flow_models.go     # Flow event data models
├── indexer/
│   ├── indexer.go         # Indexing logic (for EVM contracts)
│   ├── parser.go          # Event parsing
│   └── rpc.go             # RPC client
├── .golangci.yml          # Linter configuration
├── config.yaml            # Runtime config (gitignored)
├── config-example.yaml    # Config template
└── networks.json          # Network contract addresses
```

### Data Flow

1. **Initialization**: Load config, connect to database, create Flow gRPC client
2. **Sync State**: Check last indexed block from database
3. **Block Processing**: Query events in batches (default: 200 blocks)
4. **Event Parsing**: Parse Cadence event data into Go structs
5. **Storage**: Store events in PostgreSQL with deduplication
6. **State Update**: Update sync state after each successful batch
7. **Continuous Loop**: Poll for new blocks every 2 seconds

### Database Models

- `flow_market_createds` - Market creation events
- `flow_bet_placeds` - Bet placement events
- `flow_market_resolveds` - Market resolution events
- `flow_winnings_claimeds` - Winnings claim events
- `flow_yield_depositeds` - Yield deposit events
- `flow_yield_withdrawns` - Yield withdrawal events
- `flow_sync_states` - Indexer sync state per contract

## Code Quality

The project uses `golangci-lint` with auto-fixable linters:
- `gofmt`, `gofumpt` - Code formatting
- `goimports`, `gci` - Import management
- `whitespace` - Whitespace consistency
- `gosimple`, `staticcheck` - Code quality
- `errcheck` - Error handling checks
- `revive`, `stylecheck` - Style consistency

## Flow Network Endpoints

### Testnet
```
access.devnet.nodes.onflow.org:9000
```

### Mainnet
```
access.mainnet.nodes.onflow.org:9000
```

## Troubleshooting

### Cannot find golangci-lint
Add Go bin to PATH:
```bash
export PATH=$PATH:$HOME/go/bin
```

### Database connection failed
Ensure PostgreSQL is running and credentials are correct:
```bash
psql -U your_username -d trixy-flow-indexer
```

### Flow client connection issues
Verify the gRPC endpoint is accessible:
```bash
grpcurl access.devnet.nodes.onflow.org:9000 list
```

## License

MIT
