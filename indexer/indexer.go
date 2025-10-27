// Package indexer provides blockchain indexing functionality.
package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/evaafi/go-indexer/config"
)

var (
	Shutdown = make(chan struct{})
	WG       sync.WaitGroup
)

func RunIndexer(ctx context.Context, cfg config.Config) {
	rpcClient, err := NewRPCClient(cfg.RPCEndpoint)
	if err != nil {
		fmt.Printf("Failed to create RPC client: %v\n", err)
		return
	}
	defer rpcClient.Close()

	for _, contract := range config.Contracts {
		WG.Add(1)

		go indexContract(ctx, cfg, rpcClient, contract)
	}
}

func indexContract(ctx context.Context, cfg config.Config, rpcClient *RPCClient, contract config.Contract) {
	defer WG.Done()

	db, err := config.GetDBInstance()
	if err != nil {
		fmt.Printf("Failed to get DB instance for %s: %v\n", contract.Name, err)
		return
	}

	fmt.Printf("Starting indexer for contract %s (%s)\n", contract.Name, contract.Address)

	for {
		select {
		case <-ctx.Done():
			return
		case <-Shutdown:
			return
		default:
		}

		var state config.SyncState

		err := db.Where("contract_address = ?", contract.Address).First(&state).Error
		if err != nil {
			fmt.Printf("Error getting sync state for %s: %v\n", contract.Name, err)
			time.Sleep(5 * time.Second)

			continue
		}

		latestBlock, err := rpcClient.GetLatestBlockNumber(ctx)
		if err != nil {
			fmt.Printf("Error getting latest block: %v\n", err)
			time.Sleep(5 * time.Second)

			continue
		}

		if uint64(state.LastBlock) >= latestBlock {
			time.Sleep(5 * time.Second)
			continue
		}

		fromBlock := uint64(state.LastBlock) + 1

		toBlock := fromBlock + uint64(cfg.BlockBatchSize) - 1
		if toBlock > latestBlock {
			toBlock = latestBlock
		}

		fmt.Printf("[%s] Processing blocks %d to %d (latest: %d)\n",
			contract.Name, fromBlock, toBlock, latestBlock)

		if err := processBlockRange(ctx, db, rpcClient, contract, fromBlock, toBlock); err != nil {
			fmt.Printf("Error processing block range for %s: %v\n", contract.Name, err)
			time.Sleep(5 * time.Second)

			continue
		}

		state.LastBlock = int64(toBlock)
		if err := db.Save(&state).Error; err != nil {
			fmt.Printf("Error updating sync state for %s: %v\n", contract.Name, err)
			time.Sleep(5 * time.Second)

			continue
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func processBlockRange(ctx context.Context, db *gorm.DB, rpcClient *RPCClient, contract config.Contract, fromBlock, toBlock uint64) error {
	logs, err := rpcClient.GetLogs(ctx, contract.Address, fromBlock, toBlock)
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	if len(logs) == 0 {
		return nil
	}

	fmt.Printf("[%s] Found %d events in blocks %d-%d\n", contract.Name, len(logs), fromBlock, toBlock)

	blockTimestamps := make(map[uint64]uint64)

	var entities []interface{}

	for _, log := range logs {
		blockNum := log.BlockNumber
		timestamp, ok := blockTimestamps[blockNum]

		if !ok {
			header, err := rpcClient.GetBlockWithTimestamp(ctx, blockNum)
			if err != nil {
				fmt.Printf("Warning: failed to get block %d timestamp: %v\n", blockNum, err)

				timestamp = uint64(time.Now().Unix())
			} else {
				timestamp = header.Time
			}

			blockTimestamps[blockNum] = timestamp
		}

		entity, err := ParseLog(log, contract.Address, timestamp)
		if err != nil {
			fmt.Printf("Warning: failed to parse log at block %d, tx %s: %v\n",
				log.BlockNumber, log.TxHash.Hex(), err)
			continue
		}

		entities = append(entities, entity)
	}

	if len(entities) == 0 {
		return nil
	}

	return storeEntities(db, entities)
}

func storeEntities(db *gorm.DB, entities []interface{}) error {
	var (
		betPlaced       []*config.BetPlaced
		marketCreated   []*config.MarketCreated
		marketResolved  []*config.MarketResolved
		winningsClaimed []*config.WinningsClaimed
		autoDeposit     []*config.AutoDepositExecuted
		autoWithdraw    []*config.AutoWithdrawExecuted
		ownership       []*config.OwnershipTransferred
		paused          []*config.Paused
		protocolReg     []*config.ProtocolRegistered
		protocolUpd     []*config.ProtocolUpdated
		unpaused        []*config.Unpaused
	)

	for _, entity := range entities {
		switch e := entity.(type) {
		case *config.BetPlaced:
			betPlaced = append(betPlaced, e)
		case *config.MarketCreated:
			marketCreated = append(marketCreated, e)
		case *config.MarketResolved:
			marketResolved = append(marketResolved, e)
		case *config.WinningsClaimed:
			winningsClaimed = append(winningsClaimed, e)
		case *config.AutoDepositExecuted:
			autoDeposit = append(autoDeposit, e)
		case *config.AutoWithdrawExecuted:
			autoWithdraw = append(autoWithdraw, e)
		case *config.OwnershipTransferred:
			ownership = append(ownership, e)
		case *config.Paused:
			paused = append(paused, e)
		case *config.ProtocolRegistered:
			protocolReg = append(protocolReg, e)
		case *config.ProtocolUpdated:
			protocolUpd = append(protocolUpd, e)
		case *config.Unpaused:
			unpaused = append(unpaused, e)
		}
	}

	insertSlice := func(slice interface{}) error {
		if slice == nil {
			return nil
		}

		return db.Clauses(clause.OnConflict{DoNothing: true}).Create(slice).Error
	}

	if len(betPlaced) > 0 {
		if err := insertSlice(&betPlaced); err != nil {
			return fmt.Errorf("failed to insert BetPlaced: %w", err)
		}

		fmt.Printf("Inserted %d BetPlaced events\n", len(betPlaced))
	}

	if len(marketCreated) > 0 {
		if err := insertSlice(&marketCreated); err != nil {
			return fmt.Errorf("failed to insert MarketCreated: %w", err)
		}

		fmt.Printf("Inserted %d MarketCreated events\n", len(marketCreated))
	}

	if len(marketResolved) > 0 {
		if err := insertSlice(&marketResolved); err != nil {
			return fmt.Errorf("failed to insert MarketResolved: %w", err)
		}

		fmt.Printf("Inserted %d MarketResolved events\n", len(marketResolved))
	}

	if len(winningsClaimed) > 0 {
		if err := insertSlice(&winningsClaimed); err != nil {
			return fmt.Errorf("failed to insert WinningsClaimed: %w", err)
		}

		fmt.Printf("Inserted %d WinningsClaimed events\n", len(winningsClaimed))
	}

	if len(autoDeposit) > 0 {
		if err := insertSlice(&autoDeposit); err != nil {
			return fmt.Errorf("failed to insert AutoDepositExecuted: %w", err)
		}

		fmt.Printf("Inserted %d AutoDepositExecuted events\n", len(autoDeposit))
	}

	if len(autoWithdraw) > 0 {
		if err := insertSlice(&autoWithdraw); err != nil {
			return fmt.Errorf("failed to insert AutoWithdrawExecuted: %w", err)
		}

		fmt.Printf("Inserted %d AutoWithdrawExecuted events\n", len(autoWithdraw))
	}

	if len(ownership) > 0 {
		if err := insertSlice(&ownership); err != nil {
			return fmt.Errorf("failed to insert OwnershipTransferred: %w", err)
		}

		fmt.Printf("Inserted %d OwnershipTransferred events\n", len(ownership))
	}

	if len(paused) > 0 {
		if err := insertSlice(&paused); err != nil {
			return fmt.Errorf("failed to insert Paused: %w", err)
		}

		fmt.Printf("Inserted %d Paused events\n", len(paused))
	}

	if len(protocolReg) > 0 {
		if err := insertSlice(&protocolReg); err != nil {
			return fmt.Errorf("failed to insert ProtocolRegistered: %w", err)
		}

		fmt.Printf("Inserted %d ProtocolRegistered events\n", len(protocolReg))
	}

	if len(protocolUpd) > 0 {
		if err := insertSlice(&protocolUpd); err != nil {
			return fmt.Errorf("failed to insert ProtocolUpdated: %w", err)
		}

		fmt.Printf("Inserted %d ProtocolUpdated events\n", len(protocolUpd))
	}

	if len(unpaused) > 0 {
		if err := insertSlice(&unpaused); err != nil {
			return fmt.Errorf("failed to insert Unpaused: %w", err)
		}

		fmt.Printf("Inserted %d Unpaused events\n", len(unpaused))
	}

	return nil
}

func SaveQueue() error {
	return nil
}
