package models

import "time"

type TransactionType string

const (
	TypeWithdrawal TransactionType = "withdrawal"
	TypeDeposit    TransactionType = "deposit"
	TypeTransfer   TransactionType = "transfer"
)

// Transaction represents a standardized financial transaction parsed from CSV/XLSX
type Transaction struct {
	ID                 string          `json:"id"`
	Date               time.Time       `json:"date"`
	Type               TransactionType `json:"type"`
	Amount             string          `json:"amount"` // Use string to prevent float precision issues
	Description        string          `json:"description"`
	Counterparty       string          `json:"counterparty"`
	Category           string          `json:"category"`
	PaymentMethod      string          `json:"paymentMethod"`
	Status             string          `json:"status"`
	TransactionID      string          `json:"transactionId"`
	CategoryId         string          `json:"categoryId"`
	AssetAccount       string          `json:"assetAccount"`
	AssetId            string          `json:"assetId"`
	OpposingAccount    string          `json:"opposingAccount"`
	OpposingId         string          `json:"opposingId"`
	Notes              string          `json:"notes"`
	Tags               []string        `json:"tags"`
	Ignore             bool            `json:"ignore"`
	RawData            string          `json:"rawData"`
	Source             string          `json:"source"`
	IsOutflow          bool            `json:"isOutflow"` // JSON encoded raw row for reference
}
