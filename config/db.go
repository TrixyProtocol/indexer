// Package config provides database models and utilities.
package config

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DBInstance *gorm.DB
	dbOnce     sync.Once
	CFG        Config
)

func GetDBInstance() (*gorm.DB, error) {
	var err error

	dbOnce.Do(func() {
		var dsn string
		if CFG.DBPass != "" {
			dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=prefer",
				CFG.DBHost, CFG.DBPort, CFG.DBUser, CFG.DBPass, CFG.DBName)
		} else {
			dsn = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=prefer",
				CFG.DBHost, CFG.DBPort, CFG.DBUser, CFG.DBName)
		}

		DBInstance, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.New(
				log.New(os.Stdout, "\r\n", log.LstdFlags),
				logger.Config{
					SlowThreshold: 0,
					LogLevel:      logger.Warn,
					Colorful:      true,
				},
			),
		})
	})

	return DBInstance, err
}

func GetTableName(db *gorm.DB, model interface{}) string {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		// Handle the error, e.g., log it or return it
		return ""
	}

	return stmt.Schema.Table
}

type Principals map[BigInt]BigInt

func (p Principals) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *Principals) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("unable to scan Principals, src is %T", src)
	}

	return json.Unmarshal(bytes, p)
}

func (b BigInt) MarshalJSON() ([]byte, error) {
	if b.Int == nil {
		return []byte("null"), nil
	}

	return []byte(`"` + b.String() + `"`), nil
}

func (b *BigInt) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		b.Int = big.NewInt(0)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	bi, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return fmt.Errorf("BigInt: cannot parse %q", s)
	}

	b.Int = bi

	return nil
}

type BetPlaced struct {
	ID              string `gorm:"primaryKey;column:id"`
	MarketID        BigInt `gorm:"column:market_id;type:NUMERIC;not null;index"`
	User            string `gorm:"column:user;not null;index"`
	Position        bool   `gorm:"column:position;not null"`
	Amount          BigInt `gorm:"column:amount;type:NUMERIC;not null"`
	Shares          BigInt `gorm:"column:shares;type:NUMERIC;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type MarketCreated struct {
	ID              string `gorm:"primaryKey;column:id"`
	MarketID        BigInt `gorm:"column:market_id;type:NUMERIC;not null;index"`
	Question        string `gorm:"column:question;not null"`
	EndTime         BigInt `gorm:"column:end_time;type:NUMERIC;not null"`
	TokenAddress    string `gorm:"column:token_address;not null"`
	VaultAddress    string `gorm:"column:vault_address;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type MarketResolved struct {
	ID              string `gorm:"primaryKey;column:id"`
	MarketID        BigInt `gorm:"column:market_id;type:NUMERIC;not null;index"`
	Outcome         bool   `gorm:"column:outcome;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type WinningsClaimed struct {
	ID              string `gorm:"primaryKey;column:id"`
	MarketID        BigInt `gorm:"column:market_id;type:NUMERIC;not null;index"`
	User            string `gorm:"column:user;not null;index"`
	WinningAmount   BigInt `gorm:"column:winning_amount;type:NUMERIC;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type AutoDepositExecuted struct {
	ID              string `gorm:"primaryKey;column:id"`
	User            string `gorm:"column:user;not null;index"`
	Protocol        string `gorm:"column:protocol;not null"`
	Amount          BigInt `gorm:"column:amount;type:NUMERIC;not null"`
	Success         bool   `gorm:"column:success;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type AutoWithdrawExecuted struct {
	ID              string `gorm:"primaryKey;column:id"`
	User            string `gorm:"column:user;not null;index"`
	Protocol        string `gorm:"column:protocol;not null"`
	Amount          BigInt `gorm:"column:amount;type:NUMERIC;not null"`
	Success         bool   `gorm:"column:success;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type OwnershipTransferred struct {
	ID              string `gorm:"primaryKey;column:id"`
	PreviousOwner   string `gorm:"column:previous_owner;not null"`
	NewOwner        string `gorm:"column:new_owner;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type Paused struct {
	ID              string `gorm:"primaryKey;column:id"`
	Account         string `gorm:"column:account;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type ProtocolRegistered struct {
	ID              string `gorm:"primaryKey;column:id"`
	ProtocolType    int    `gorm:"column:protocol_type;not null"`
	ProtocolAddress string `gorm:"column:protocol_address;not null;index"`
	Name            string `gorm:"column:name;not null"`
	RiskLevel       int    `gorm:"column:risk_level;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type ProtocolUpdated struct {
	ID              string `gorm:"primaryKey;column:id"`
	ProtocolAddress string `gorm:"column:protocol_address;not null;index"`
	NewApy          BigInt `gorm:"column:new_apy;type:NUMERIC;not null"`
	NewTvl          BigInt `gorm:"column:new_tvl;type:NUMERIC;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type Unpaused struct {
	ID              string `gorm:"primaryKey;column:id"`
	Account         string `gorm:"column:account;not null"`
	BlockNumber     BigInt `gorm:"column:block_number;type:NUMERIC;not null"`
	BlockTimestamp  BigInt `gorm:"column:block_timestamp;type:NUMERIC;not null"`
	TransactionHash string `gorm:"column:transaction_hash;not null;index"`
}

type BigInt struct {
	*big.Int
}

func (b BigInt) Value() (driver.Value, error) {
	if b.Int == nil {
		return "0", nil
	}

	return b.String(), nil
}

func (b *BigInt) Scan(value interface{}) error {
	if value == nil {
		b.Int = big.NewInt(0)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		s := string(v)
		i, ok := new(big.Int).SetString(s, 10)

		if !ok {
			return fmt.Errorf("cannot convert %s to big.Int", s)
		}

		b.Int = i
	case string:
		i, ok := new(big.Int).SetString(v, 10)
		if !ok {
			return fmt.Errorf("cannot convert %s to big.Int", v)
		}

		b.Int = i
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}

	return nil
}

type SyncState struct {
	ContractAddress string `gorm:"primaryKey;column:contract_address"`
	ContractName    string `gorm:"column:contract_name;not null"`
	LastBlock       int64  `gorm:"column:last_block;not null"`
	LastBlockHash   string `gorm:"column:last_block_hash"`
}

func EnsureInitialSyncStateData(db *gorm.DB) {
	if len(Contracts) == 0 {
		fmt.Println("Warning: No contracts loaded, skipping sync state initialization")
		return
	}

	for _, contract := range Contracts {
		var existing SyncState

		err := db.First(&existing, "contract_address = ?", contract.Address).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				data := SyncState{
					ContractAddress: contract.Address,
					ContractName:    contract.Name,
					LastBlock:       contract.StartBlock,
					LastBlockHash:   "",
				}
				if err := db.Create(&data).Error; err != nil {
					fmt.Printf("Failed to insert initial data for contract %s: %v\n", contract.Name, err)
				} else {
					fmt.Printf("Inserted initial data for contract %s (start block: %d)\n", contract.Name, contract.StartBlock)
				}
			} else {
				fmt.Printf("Error checking existing record for %s: %v\n", contract.Name, err)
			}
		} else {
			fmt.Printf("Sync state already exists for contract %s (current block: %d)\n", contract.Name, existing.LastBlock)
		}
	}
}
