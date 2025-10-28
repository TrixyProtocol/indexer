// Package config provides Flow blockchain data models.
package config

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type FlowMarketCreated struct {
	ID             uint        `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	MarketID       uint64      `gorm:"column:market_id;index" json:"marketId"`
	Question       string      `gorm:"column:question" json:"question"`
	EndTime        string      `gorm:"column:end_time" json:"endTime"`
	Protocols      StringArray `gorm:"column:protocols;type:text[]" json:"protocols"`
	Options        StringArray `gorm:"column:options;type:text[]" json:"options"`
	YieldProtocol  string      `gorm:"column:yield_protocol" json:"yieldProtocol"`
	Creator        string      `gorm:"column:creator;index" json:"creator"`
	BlockHeight    uint64      `gorm:"column:block_height;index" json:"blockHeight"`
	BlockTimestamp int64       `gorm:"column:block_timestamp;index" json:"blockTimestamp"`
	TransactionID  string      `gorm:"column:transaction_id;index" json:"transactionId"`
	EventIndex     uint32      `gorm:"column:event_index" json:"eventIndex"`
	CreatedAt      time.Time   `gorm:"autoCreateTime" json:"createdAt"`
}

func (FlowMarketCreated) TableName() string {
	return "flow_market_createds"
}

type FlowBetPlaced struct {
	ID             uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	MarketID       uint64    `gorm:"column:market_id;index" json:"marketId"`
	User           string    `gorm:"column:user;index" json:"user"`
	SelectedOption string    `gorm:"column:selected_option" json:"selectedOption"`
	ProtocolIndex  uint32    `gorm:"column:protocol_index" json:"protocolIndex"`
	Amount         string    `gorm:"column:amount" json:"amount"`
	BlockHeight    uint64    `gorm:"column:block_height;index" json:"blockHeight"`
	BlockTimestamp int64     `gorm:"column:block_timestamp;index" json:"blockTimestamp"`
	TransactionID  string    `gorm:"column:transaction_id;index" json:"transactionId"`
	EventIndex     uint32    `gorm:"column:event_index" json:"eventIndex"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (FlowBetPlaced) TableName() string {
	return "flow_bet_placeds"
}

type FlowMarketResolved struct {
	ID             uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	MarketID       uint64    `gorm:"column:market_id;index" json:"marketId"`
	WinningOption  string    `gorm:"column:winning_option" json:"winningOption"`
	FinalAPYs      JSONMap   `gorm:"column:final_apys;type:jsonb" json:"finalAPYs"`
	ResolvedAt     string    `gorm:"column:resolved_at" json:"resolvedAt"`
	BlockHeight    uint64    `gorm:"column:block_height;index" json:"blockHeight"`
	BlockTimestamp int64     `gorm:"column:block_timestamp;index" json:"blockTimestamp"`
	TransactionID  string    `gorm:"column:transaction_id;index" json:"transactionId"`
	EventIndex     uint32    `gorm:"column:event_index" json:"eventIndex"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (FlowMarketResolved) TableName() string {
	return "flow_market_resolveds"
}

type FlowWinningsClaimed struct {
	ID             uint   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	MarketID       uint64 `gorm:"column:market_id;index" json:"marketId"`
	User           string `gorm:"column:user;index" json:"user"`
	Payout         string `gorm:"column:payout" json:"payout"`
	BlockHeight    uint64 `gorm:"column:block_height;index" json:"blockHeight"`
	BlockTimestamp int64  `gorm:"column:block_timestamp;index" json:"blockTimestamp"`
	TransactionID  string `gorm:"column:transaction_id;index" json:"transactionId"`
	EventIndex     uint32 `gorm:"column:event_index" json:"eventIndex"`
}

func (FlowWinningsClaimed) TableName() string {
	return "flow_winnings_claimeds"
}

type FlowYieldDeposited struct {
	ID             uint   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserAddress    string `gorm:"column:user_address;index" json:"userAddress"`
	ProtocolName   string `gorm:"column:protocol_name;index" json:"protocolName"`
	Amount         string `gorm:"column:amount" json:"amount"`
	PositionID     string `gorm:"column:position_id" json:"positionId"`
	BlockHeight    uint64 `gorm:"column:block_height;index" json:"blockHeight"`
	BlockTimestamp int64  `gorm:"column:block_timestamp;index" json:"blockTimestamp"`
	TransactionID  string `gorm:"column:transaction_id;index" json:"transactionId"`
	EventIndex     uint32 `gorm:"column:event_index" json:"eventIndex"`
}

func (FlowYieldDeposited) TableName() string {
	return "flow_yield_depositeds"
}

type FlowYieldWithdrawn struct {
	ID             uint   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	MarketID       uint64 `gorm:"column:market_id;index" json:"marketId"`
	Protocol       string `gorm:"column:protocol" json:"protocol"`
	Amount         string `gorm:"column:amount" json:"amount"`
	YieldEarned    string `gorm:"column:yield_earned" json:"yieldEarned"`
	BlockHeight    uint64 `gorm:"column:block_height;index" json:"blockHeight"`
	BlockTimestamp int64  `gorm:"column:block_timestamp;index" json:"blockTimestamp"`
	TransactionID  string `gorm:"column:transaction_id;index" json:"transactionId"`
	EventIndex     uint32 `gorm:"column:event_index" json:"eventIndex"`
}

func (FlowYieldWithdrawn) TableName() string {
	return "flow_yield_withdrawns"
}

type FlowSyncState struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ContractAddress string    `gorm:"uniqueIndex;column:contract_address" json:"contractAddress"`
	ContractName    string    `gorm:"column:contract_name" json:"contractName"`
	Network         string    `gorm:"column:network" json:"network"`
	LastBlockHeight uint64    `gorm:"column:last_block_height" json:"lastBlockHeight"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (FlowSyncState) TableName() string {
	return "flow_sync_states"
}

type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	return "{" + strings.Join(a, ",") + "}", nil
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		s := string(v)
		s = strings.TrimPrefix(s, "{")
		s = strings.TrimSuffix(s, "}")

		if s == "" {
			*a = []string{}
			return nil
		}

		*a = strings.Split(s, ",")

		return nil
	case string:
		s := strings.TrimPrefix(v, "{")
		s = strings.TrimSuffix(s, "}")

		if s == "" {
			*a = []string{}
			return nil
		}

		*a = strings.Split(s, ",")

		return nil
	default:
		return fmt.Errorf("failed to scan StringArray value: %v", value)
	}
}

type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}

	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONMap value: %v", value)
	}

	return json.Unmarshal(bytes, j)
}
