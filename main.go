package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/access/grpc"
	"gorm.io/gorm"

	"github.com/evaafi/go-indexer/config"
)

const (
	batchSize    = 200
	pollInterval = 2 * time.Second
)

func main() {
	fmt.Println("Trixy Protocol Flow Indexer")
	fmt.Println("================================")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config.CFG = cfg

	db, err := config.GetDBInstance()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	ctx := context.Background()

	flowClient, err := grpc.NewClient(cfg.RPCEndpoint)
	if err != nil {
		log.Fatalf("Failed to create Flow client: %v", err)
	}

	defer func() {
		if err := flowClient.Close(); err != nil {
			log.Printf("Failed to close Flow client: %v", err)
		}
	}()

	if err := config.LoadNetworks(cfg.NetworksFile, cfg.Network); err != nil {
		log.Fatalf("Failed to load networks: %v", err)
	}

	if len(config.Contracts) == 0 {
		log.Fatal("No contracts configured")
	}

	contract := config.Contracts[0]
	contractAddress := strings.TrimPrefix(contract.Address, "0x")

	fmt.Printf("Contract: %s at %s\n", contract.Name, contract.Address)
	fmt.Printf("Network: %s\n", cfg.Network)
	fmt.Printf("RPC: %s\n\n", cfg.RPCEndpoint)

	var syncState config.FlowSyncState

	result := db.Where("contract_address = ?", contract.Address).First(&syncState)
	if result.Error != nil {
		syncState = config.FlowSyncState{
			ContractAddress: contract.Address,
			ContractName:    contract.Name,
			Network:         cfg.Network,
			LastBlockHeight: uint64(contract.StartBlock),
		}
		if err := db.Create(&syncState).Error; err != nil {
			log.Fatalf("Failed to create sync state: %v", err)
		}

		fmt.Printf("Created new sync state starting at block %d\n\n", contract.StartBlock)
	} else {
		fmt.Printf("Resuming from block %d\n\n", syncState.LastBlockHeight)
	}

	latestBlock, err := flowClient.GetLatestBlock(ctx, true)
	if err != nil {
		log.Fatalf("Failed to get latest block: %v", err)
	}

	fmt.Printf("Latest network block: %d\n", latestBlock.Height)
	fmt.Printf("Blocks to index: %d\n\n", latestBlock.Height-syncState.LastBlockHeight)

	totalEvents := 0

	fmt.Println("🔄 Starting continuous indexing mode...")

	for {
		latestBlock, err := flowClient.GetLatestBlock(ctx, true)
		if err != nil {
			log.Printf("Error getting latest block: %v", err)
			time.Sleep(pollInterval)

			continue
		}

		if err := db.Where("contract_address = ?", contract.Address).First(&syncState).Error; err != nil {
			log.Printf("Error loading sync state: %v", err)
			time.Sleep(pollInterval)

			continue
		}

		currentBlock := syncState.LastBlockHeight

		if currentBlock < latestBlock.Height {
			fmt.Printf("📈 Latest block: %d, Current: %d, Gap: %d blocks\n", latestBlock.Height, currentBlock, latestBlock.Height-currentBlock)

			for currentBlock < latestBlock.Height {
				endBlock := currentBlock + batchSize
				if endBlock > latestBlock.Height {
					endBlock = latestBlock.Height
				}

				fmt.Printf("⏳ Indexing blocks %d to %d...\n", currentBlock, endBlock)

				events := indexBatch(ctx, db, flowClient, contractAddress, currentBlock, endBlock)
				totalEvents += events

				if events > 0 {
					fmt.Printf("✓ Indexed %d events\n\n", events)
				} else {
					fmt.Printf("  No events found\n\n")
				}

				syncState.LastBlockHeight = endBlock
				if err := db.Save(&syncState).Error; err != nil {
					log.Printf("Warning: Failed to update sync state: %v", err)
				}

				currentBlock = endBlock + 1
			}

			fmt.Printf("✅ Caught up! Total events: %d\n", totalEvents)
			showTableCounts(db)
			fmt.Printf("\n⏰ Waiting %v for new blocks...\n\n", pollInterval)
		} else {
			fmt.Printf("⏰ Up to date at block %d. Checking again in %v...\n", currentBlock, pollInterval)
		}

		time.Sleep(pollInterval)
	}
}

func indexBatch(ctx context.Context, db *gorm.DB, flowClient *grpc.Client, contractAddress string, startBlock, endBlock uint64) int {
	total := 0

	eventTypes := map[string]string{
		"MarketCreated":   fmt.Sprintf("A.%s.TrixyEvents.MarketCreated", contractAddress),
		"BetPlaced":       fmt.Sprintf("A.%s.TrixyEvents.BetPlaced", contractAddress),
		"MarketResolved":  fmt.Sprintf("A.%s.TrixyEvents.MarketResolved", contractAddress),
		"WinningsClaimed": fmt.Sprintf("A.%s.TrixyEvents.WinningsClaimed", contractAddress),
		"YieldDeposited":  fmt.Sprintf("A.%s.TrixyEvents.YieldDeposited", contractAddress),
		"YieldWithdrawn":  fmt.Sprintf("A.%s.TrixyEvents.YieldWithdrawn", contractAddress),
	}

	for name, eventType := range eventTypes {
		events, err := flowClient.GetEventsForHeightRange(ctx, eventType, startBlock, endBlock)
		if err != nil {
			log.Printf("Error querying %s: %v", name, err)
			continue
		}

		for _, blockEvents := range events {
			for _, event := range blockEvents.Events {
				if err := storeEvent(ctx, db, flowClient, event, blockEvents.Height, name); err != nil {
					if !strings.Contains(err.Error(), "duplicate key") {
						log.Printf("Error storing %s event: %v", name, err)
					}

					continue
				}

				total++
			}
		}
	}

	return total
}

func storeEvent(ctx context.Context, db *gorm.DB, flowClient *grpc.Client, event flow.Event, blockHeight uint64, eventName string) error {
	block, err := flowClient.GetBlockByHeight(ctx, blockHeight)
	if err != nil {
		return err
	}

	fields := cadence.FieldsMappedByName(event.Value)

	switch eventName {
	case "MarketCreated":
		options := []string{}

		optionsField := fields["options"]
		if optionsField == nil {
			optionsField = fields["protocols"]
		}

		if optionsField != nil {
			for _, p := range optionsField.(cadence.Array).Values {
				options = append(options, string(p.(cadence.String)))
			}
		}

		endTimeUFix64 := fields["endTime"].(cadence.UFix64)
		endTimeInt := int64(endTimeUFix64 / 100000000)

		yieldProtocol := ""
		if yieldProtoField := fields["yieldProtocol"]; yieldProtoField != nil {
			yieldProtocol = string(yieldProtoField.(cadence.String))
		}

		return db.Create(&config.FlowMarketCreated{
			MarketID:       uint64(fields["marketId"].(cadence.UInt64)),
			Question:       string(fields["question"].(cadence.String)),
			EndTime:        fmt.Sprintf("%d", endTimeInt),
			Protocols:      options,
			Options:        options,
			YieldProtocol:  yieldProtocol,
			Creator:        fields["creator"].(cadence.Address).String(),
			BlockHeight:    blockHeight,
			BlockTimestamp: block.Timestamp.Unix(),
			TransactionID:  event.TransactionID.String(),
			EventIndex:     uint32(event.EventIndex),
		}).Error

	case "BetPlaced":
		// Extract protocolIndex, default to 0 if not present
		protocolIndex := uint32(0)
		if protocolIndexField := fields["protocolIndex"]; protocolIndexField != nil {
			protocolIndex = uint32(protocolIndexField.(cadence.UInt32))
		}

		return db.Create(&config.FlowBetPlaced{
			MarketID:       uint64(fields["marketId"].(cadence.UInt64)),
			User:           fields["user"].(cadence.Address).String(),
			SelectedOption: string(fields["selectedOption"].(cadence.String)),
			ProtocolIndex:  protocolIndex,
			Amount:         fields["amount"].(cadence.UFix64).String(),
			BlockHeight:    blockHeight,
			BlockTimestamp: block.Timestamp.Unix(),
			TransactionID:  event.TransactionID.String(),
			EventIndex:     uint32(event.EventIndex),
		}).Error

	case "MarketResolved":
		apysDict := fields["finalAPYs"].(cadence.Dictionary)
		finalAPYs := make(map[string]interface{})

		for _, pair := range apysDict.Pairs {
			key := string(pair.Key.(cadence.String))
			value := pair.Value.(cadence.UFix64).String()
			finalAPYs[key] = value
		}

		return db.Create(&config.FlowMarketResolved{
			MarketID:       uint64(fields["marketId"].(cadence.UInt64)),
			WinningOption:  string(fields["winningOption"].(cadence.String)),
			FinalAPYs:      finalAPYs,
			ResolvedAt:     fields["resolvedAt"].(cadence.UFix64).String(),
			BlockHeight:    blockHeight,
			BlockTimestamp: block.Timestamp.Unix(),
			TransactionID:  event.TransactionID.String(),
			EventIndex:     uint32(event.EventIndex),
		}).Error

	case "WinningsClaimed":
		return db.Create(&config.FlowWinningsClaimed{
			MarketID:       uint64(fields["marketId"].(cadence.UInt64)),
			User:           fields["user"].(cadence.Address).String(),
			Payout:         fields["payout"].(cadence.UFix64).String(),
			BlockHeight:    blockHeight,
			BlockTimestamp: block.Timestamp.Unix(),
			TransactionID:  event.TransactionID.String(),
			EventIndex:     uint32(event.EventIndex),
		}).Error

	case "YieldDeposited":
		// Extract fields with fallbacks for different field names
		userAddress := ""
		if userField := fields["user"]; userField != nil {
			userAddress = userField.(cadence.Address).String()
		} else if userAddrField := fields["userAddress"]; userAddrField != nil {
			userAddress = userAddrField.(cadence.Address).String()
		}

		protocolName := ""
		if protocolField := fields["protocol"]; protocolField != nil {
			protocolName = string(protocolField.(cadence.String))
		} else if protoNameField := fields["protocolName"]; protoNameField != nil {
			protocolName = string(protoNameField.(cadence.String))
		}

		positionID := ""
		if posIDField := fields["positionId"]; posIDField != nil {
			positionID = string(posIDField.(cadence.String))
		} else if marketIDField := fields["marketId"]; marketIDField != nil {
			positionID = fmt.Sprintf("%d", uint64(marketIDField.(cadence.UInt64)))
		}

		return db.Create(&config.FlowYieldDeposited{
			UserAddress:    userAddress,
			ProtocolName:   protocolName,
			Amount:         fields["amount"].(cadence.UFix64).String(),
			PositionID:     positionID,
			BlockHeight:    blockHeight,
			BlockTimestamp: block.Timestamp.Unix(),
			TransactionID:  event.TransactionID.String(),
			EventIndex:     uint32(event.EventIndex),
		}).Error

	case "YieldWithdrawn":
		return db.Create(&config.FlowYieldWithdrawn{
			MarketID:       uint64(fields["marketId"].(cadence.UInt64)),
			Protocol:       string(fields["protocol"].(cadence.String)),
			Amount:         fields["amount"].(cadence.UFix64).String(),
			YieldEarned:    fields["yieldEarned"].(cadence.UFix64).String(),
			BlockHeight:    blockHeight,
			BlockTimestamp: block.Timestamp.Unix(),
			TransactionID:  event.TransactionID.String(),
			EventIndex:     uint32(event.EventIndex),
		}).Error
	}

	return nil
}

func showTableCounts(db *gorm.DB) {
	fmt.Println("📊 Database Summary:")

	tables := []struct {
		name  string
		model interface{}
	}{
		{"flow_market_createds", &config.FlowMarketCreated{}},
		{"flow_bet_placeds", &config.FlowBetPlaced{}},
		{"flow_market_resolveds", &config.FlowMarketResolved{}},
		{"flow_winnings_claimeds", &config.FlowWinningsClaimed{}},
		{"flow_yield_depositeds", &config.FlowYieldDeposited{}},
		{"flow_yield_withdrawns", &config.FlowYieldWithdrawn{}},
	}

	for _, table := range tables {
		var count int64

		db.Model(table.model).Count(&count)
		fmt.Printf("  - %-25s %d rows\n", table.name+":", count)
	}
}
