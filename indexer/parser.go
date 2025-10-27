// Package indexer provides event log parsing.
package indexer

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/evaafi/go-indexer/config"
)

var (
	BetPlacedSignature       common.Hash
	MarketCreatedSignature   common.Hash
	MarketResolvedSignature  common.Hash
	WinningsClaimedSignature common.Hash

	AutoDepositExecutedSignature  common.Hash
	AutoWithdrawExecutedSignature common.Hash
	OwnershipTransferredSignature common.Hash
	PausedSignature               common.Hash
	ProtocolRegisteredSignature   common.Hash
	ProtocolUpdatedSignature      common.Hash
	UnpausedSignature             common.Hash
)

func init() {
	BetPlacedSignature = crypto.Keccak256Hash([]byte("BetPlaced(uint256,address,bool,uint256,uint256)"))
	MarketCreatedSignature = crypto.Keccak256Hash([]byte("MarketCreated(uint256,string,uint256,address,address)"))
	MarketResolvedSignature = crypto.Keccak256Hash([]byte("MarketResolved(uint256,bool)"))
	WinningsClaimedSignature = crypto.Keccak256Hash([]byte("WinningsClaimed(uint256,address,uint256)"))

	AutoDepositExecutedSignature = crypto.Keccak256Hash([]byte("AutoDepositExecuted(address,address,uint256,bool)"))
	AutoWithdrawExecutedSignature = crypto.Keccak256Hash([]byte("AutoWithdrawExecuted(address,address,uint256,bool)"))
	OwnershipTransferredSignature = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))
	PausedSignature = crypto.Keccak256Hash([]byte("Paused(address)"))
	ProtocolRegisteredSignature = crypto.Keccak256Hash([]byte("ProtocolRegistered(uint8,address,string,uint8)"))
	ProtocolUpdatedSignature = crypto.Keccak256Hash([]byte("ProtocolUpdated(address,uint256,uint256)"))
	UnpausedSignature = crypto.Keccak256Hash([]byte("Unpaused(address)"))
}

func ParseLog(log types.Log, contractAddress string, blockTimestamp uint64) (interface{}, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}

	eventSig := log.Topics[0]
	txHash := log.TxHash.Hex()
	blockNumber := config.BigInt{Int: new(big.Int).SetUint64(log.BlockNumber)}
	blockTS := config.BigInt{Int: new(big.Int).SetUint64(blockTimestamp)}

	id := fmt.Sprintf("%s-%d", txHash, log.Index)

	if strings.EqualFold(contractAddress, config.TrixyPredictionMarketContract.Address) {
		switch {
		case eventSig == BetPlacedSignature:
			return parseBetPlaced(log, id, blockNumber, blockTS, txHash)
		case eventSig == MarketCreatedSignature:
			return parseMarketCreated(log, id, blockNumber, blockTS, txHash)
		case eventSig == MarketResolvedSignature:
			return parseMarketResolved(log, id, blockNumber, blockTS, txHash)
		case eventSig == WinningsClaimedSignature:
			return parseWinningsClaimed(log, id, blockNumber, blockTS, txHash)
		}
	}

	if strings.EqualFold(contractAddress, config.ProtocolSelectorContract.Address) {
		switch {
		case eventSig == AutoDepositExecutedSignature:
			return parseAutoDepositExecuted(log, id, blockNumber, blockTS, txHash)
		case eventSig == AutoWithdrawExecutedSignature:
			return parseAutoWithdrawExecuted(log, id, blockNumber, blockTS, txHash)
		case eventSig == OwnershipTransferredSignature:
			return parseOwnershipTransferred(log, id, blockNumber, blockTS, txHash)
		case eventSig == PausedSignature:
			return parsePaused(log, id, blockNumber, blockTS, txHash)
		case eventSig == ProtocolRegisteredSignature:
			return parseProtocolRegistered(log, id, blockNumber, blockTS, txHash)
		case eventSig == ProtocolUpdatedSignature:
			return parseProtocolUpdated(log, id, blockNumber, blockTS, txHash)
		case eventSig == UnpausedSignature:
			return parseUnpaused(log, id, blockNumber, blockTS, txHash)
		}
	}

	return nil, fmt.Errorf("unknown event signature: %s for contract %s", eventSig.Hex(), contractAddress)
}

func parseBetPlaced(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.BetPlaced, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("insufficient topics for BetPlaced")
	}

	entity := &config.BetPlaced{
		ID:              id,
		MarketID:        config.BigInt{Int: new(big.Int).SetBytes(log.Topics[1].Bytes())},
		User:            common.BytesToAddress(log.Topics[2].Bytes()).Hex(),
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 96 {
		entity.Position = new(big.Int).SetBytes(log.Data[0:32]).Cmp(big.NewInt(0)) != 0
		entity.Amount = config.BigInt{Int: new(big.Int).SetBytes(log.Data[32:64])}
		entity.Shares = config.BigInt{Int: new(big.Int).SetBytes(log.Data[64:96])}
	}

	return entity, nil
}

func parseMarketCreated(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.MarketCreated, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("insufficient topics for MarketCreated")
	}

	entity := &config.MarketCreated{
		ID:              id,
		MarketID:        config.BigInt{Int: new(big.Int).SetBytes(log.Topics[1].Bytes())},
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 128 {
		offset := new(big.Int).SetBytes(log.Data[0:32]).Uint64()
		entity.EndTime = config.BigInt{Int: new(big.Int).SetBytes(log.Data[32:64])}
		entity.TokenAddress = common.BytesToAddress(log.Data[64:96]).Hex()
		entity.VaultAddress = common.BytesToAddress(log.Data[96:128]).Hex()

		if uint64(len(log.Data)) > offset+32 {
			strLen := new(big.Int).SetBytes(log.Data[offset : offset+32]).Uint64()
			if uint64(len(log.Data)) >= offset+32+strLen {
				entity.Question = string(log.Data[offset+32 : offset+32+strLen])
			}
		}
	}

	return entity, nil
}

func parseMarketResolved(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.MarketResolved, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("insufficient topics for MarketResolved")
	}

	entity := &config.MarketResolved{
		ID:              id,
		MarketID:        config.BigInt{Int: new(big.Int).SetBytes(log.Topics[1].Bytes())},
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 32 {
		entity.Outcome = new(big.Int).SetBytes(log.Data[0:32]).Cmp(big.NewInt(0)) != 0
	}

	return entity, nil
}

func parseWinningsClaimed(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.WinningsClaimed, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("insufficient topics for WinningsClaimed")
	}

	entity := &config.WinningsClaimed{
		ID:              id,
		MarketID:        config.BigInt{Int: new(big.Int).SetBytes(log.Topics[1].Bytes())},
		User:            common.BytesToAddress(log.Topics[2].Bytes()).Hex(),
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 32 {
		entity.WinningAmount = config.BigInt{Int: new(big.Int).SetBytes(log.Data[0:32])}
	}

	return entity, nil
}

func parseAutoDepositExecuted(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.AutoDepositExecuted, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("insufficient topics for AutoDepositExecuted")
	}

	entity := &config.AutoDepositExecuted{
		ID:              id,
		User:            common.BytesToAddress(log.Topics[1].Bytes()).Hex(),
		Protocol:        common.BytesToAddress(log.Topics[2].Bytes()).Hex(),
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 64 {
		entity.Amount = config.BigInt{Int: new(big.Int).SetBytes(log.Data[0:32])}
		entity.Success = new(big.Int).SetBytes(log.Data[32:64]).Cmp(big.NewInt(0)) != 0
	}

	return entity, nil
}

func parseAutoWithdrawExecuted(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.AutoWithdrawExecuted, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("insufficient topics for AutoWithdrawExecuted")
	}

	entity := &config.AutoWithdrawExecuted{
		ID:              id,
		User:            common.BytesToAddress(log.Topics[1].Bytes()).Hex(),
		Protocol:        common.BytesToAddress(log.Topics[2].Bytes()).Hex(),
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 64 {
		entity.Amount = config.BigInt{Int: new(big.Int).SetBytes(log.Data[0:32])}
		entity.Success = new(big.Int).SetBytes(log.Data[32:64]).Cmp(big.NewInt(0)) != 0
	}

	return entity, nil
}

func parseOwnershipTransferred(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.OwnershipTransferred, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("insufficient topics for OwnershipTransferred")
	}

	entity := &config.OwnershipTransferred{
		ID:              id,
		PreviousOwner:   common.BytesToAddress(log.Topics[1].Bytes()).Hex(),
		NewOwner:        common.BytesToAddress(log.Topics[2].Bytes()).Hex(),
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	return entity, nil
}

func parsePaused(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.Paused, error) {
	entity := &config.Paused{
		ID:              id,
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 32 {
		entity.Account = common.BytesToAddress(log.Data[0:32]).Hex()
	}

	return entity, nil
}

func parseProtocolRegistered(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.ProtocolRegistered, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("insufficient topics for ProtocolRegistered")
	}

	entity := &config.ProtocolRegistered{
		ID:              id,
		ProtocolType:    int(new(big.Int).SetBytes(log.Topics[1].Bytes()).Int64()),
		ProtocolAddress: common.BytesToAddress(log.Topics[2].Bytes()).Hex(),
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 64 {
		offset := new(big.Int).SetBytes(log.Data[0:32]).Uint64()
		entity.RiskLevel = int(new(big.Int).SetBytes(log.Data[32:64]).Int64())

		if uint64(len(log.Data)) > offset+32 {
			strLen := new(big.Int).SetBytes(log.Data[offset : offset+32]).Uint64()
			if uint64(len(log.Data)) >= offset+32+strLen {
				entity.Name = string(log.Data[offset+32 : offset+32+strLen])
			}
		}
	}

	return entity, nil
}

func parseProtocolUpdated(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.ProtocolUpdated, error) {
	if len(log.Topics) < 2 {
		return nil, fmt.Errorf("insufficient topics for ProtocolUpdated")
	}

	entity := &config.ProtocolUpdated{
		ID:              id,
		ProtocolAddress: common.BytesToAddress(log.Topics[1].Bytes()).Hex(),
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 64 {
		entity.NewApy = config.BigInt{Int: new(big.Int).SetBytes(log.Data[0:32])}
		entity.NewTvl = config.BigInt{Int: new(big.Int).SetBytes(log.Data[32:64])}
	}

	return entity, nil
}

func parseUnpaused(log types.Log, id string, blockNumber, blockTimestamp config.BigInt, txHash string) (*config.Unpaused, error) {
	entity := &config.Unpaused{
		ID:              id,
		BlockNumber:     blockNumber,
		BlockTimestamp:  blockTimestamp,
		TransactionHash: txHash,
	}

	if len(log.Data) >= 32 {
		entity.Account = common.BytesToAddress(log.Data[0:32]).Hex()
	}

	return entity, nil
}
