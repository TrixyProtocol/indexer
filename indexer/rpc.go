// Package indexer provides RPC client functionality.
package indexer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type RPCClient struct {
	client *ethclient.Client
}

func NewRPCClient(endpoint string) (*RPCClient, error) {
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC endpoint: %w", err)
	}

	return &RPCClient{client: client}, nil
}

func (r *RPCClient) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	header, err := r.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}

	return header.Number.Uint64(), nil
}

func (r *RPCClient) GetBlockWithTimestamp(ctx context.Context, blockNum uint64) (*types.Header, error) {
	return r.client.HeaderByNumber(ctx, big.NewInt(int64(blockNum)))
}

func (r *RPCClient) GetLogs(ctx context.Context, contractAddress string, fromBlock, toBlock uint64) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{
			common.HexToAddress(contractAddress),
		},
	}

	logs, err := r.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	return logs, nil
}

func (r *RPCClient) Close() {
	r.client.Close()
}
