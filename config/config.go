// Package config provides configuration management for the Flow indexer.
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Mode string

const (
	ModeIndexer    Mode = "indexer"
	ModeLiquidator Mode = "liquidator"
)

type DBType string

type Contract struct {
	Name       string
	Address    string
	StartBlock int64
}

var (
	TrixyPredictionMarketContract Contract
	ProtocolSelectorContract      Contract
	Contracts                     []Contract
)

type NetworkConfig map[string]map[string]struct {
	Address    string `json:"address"`
	StartBlock int64  `json:"startBlock"`
}

const (
	DBPostgres DBType = "postgres"
)

type Config struct {
	Mode                    Mode   `yaml:"mode"`
	DBType                  DBType `yaml:"dbType"`
	DBHost                  string `yaml:"dbHost"`
	DBPort                  int16  `yaml:"dbPort"`
	DBUser                  string `yaml:"dbUser"`
	DBPass                  string `yaml:"dbPass"`
	DBName                  string `yaml:"dbName"`
	RPCEndpoint             string `yaml:"rpcEndpoint"`
	Network                 string `yaml:"network"`
	NetworksFile            string `yaml:"networksFile"`
	IndexWorkers            int    `yaml:"indexWorkers"`
	ForceResyncOnEveryStart bool   `yaml:"forceResyncOnEveryStart"`
	MigrateOnStart          bool   `yaml:"migrateOnStart"`
	BlockBatchSize          int    `yaml:"blockBatchSize"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}

	if cfg.NetworksFile != "" {
		if err := LoadNetworks(cfg.NetworksFile, cfg.Network); err != nil {
			return cfg, fmt.Errorf("failed to load networks: %w", err)
		}
	}

	return cfg, nil
}

func LoadNetworks(networksFile, network string) error {
	data, err := os.ReadFile(networksFile)
	if err != nil {
		return fmt.Errorf("failed to read networks file: %w", err)
	}

	var networks NetworkConfig
	if err := json.Unmarshal(data, &networks); err != nil {
		return fmt.Errorf("failed to parse networks file: %w", err)
	}

	networkContracts, ok := networks[network]
	if !ok {
		return fmt.Errorf("network %s not found in networks file", network)
	}

	Contracts = []Contract{}

	for name, config := range networkContracts {
		contract := Contract{
			Name:       name,
			Address:    config.Address,
			StartBlock: config.StartBlock,
		}

		switch name {
		case "TrixyPredictionMarket":
			TrixyPredictionMarketContract = contract
		case "ProtocolSelector":
			ProtocolSelectorContract = contract
		}

		Contracts = append(Contracts, contract)
	}

	if len(Contracts) == 0 {
		return fmt.Errorf("no contracts found for network %s", network)
	}

	fmt.Printf("Loaded %d contracts for network %s\n", len(Contracts), network)

	for _, c := range Contracts {
		fmt.Printf("  - %s: %s (start block: %d)\n", c.Name, c.Address, c.StartBlock)
	}

	return nil
}
