package firefly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"firefly-importer/config"
	"firefly-importer/models"
)

type TransactionSplit struct {
	Type               string `json:"type"`
	Date               string `json:"date"`
	Amount             string `json:"amount"`
	Description        string `json:"description"`
	SourceId           int    `json:"source_id,omitempty"`
	SourceName         string `json:"source_name,omitempty"`
	DestinationId      int    `json:"destination_id,omitempty"`
	DestinationName    string `json:"destination_name,omitempty"`
	CategoryId         int    `json:"category_id,omitempty"`
	CategoryName       string `json:"category_name,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	Notes              string   `json:"notes,omitempty"`
	ExternalId         string   `json:"external_id,omitempty"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type TransactionStore struct {
	ErrorIfDuplicateHash bool               `json:"error_if_duplicate_hash"`
	ApplyRules           bool               `json:"apply_rules"`
	Firehooks            bool               `json:"firehooks"`
	Transactions         []TransactionSplit `json:"transactions"`
}

func SubmitTransaction(tx models.Transaction, cfg config.Config) error {
	client := &http.Client{Timeout: 60 * time.Second}

	if cfg.FireflyURL == "" || cfg.FireflyToken == "" {
		return fmt.Errorf("firefly URL or Token is not configured")
	}

	catId, _ := strconv.Atoi(tx.CategoryId)
	assetId, _ := strconv.Atoi(tx.AssetId)
	opposingId, _ := strconv.Atoi(tx.OpposingId)

	srcId := assetId
	srcName := tx.AssetAccount
	destId := opposingId
	destName := tx.OpposingAccount

	// If it's a deposit or an inflow transfer, the money comes FROM opposing TO asset
	if tx.Type == models.TypeDeposit || (tx.Type == models.TypeTransfer && !tx.IsOutflow) {
		srcId, destId = destId, srcId
		srcName, destName = destName, srcName
	}

	split := TransactionSplit{
		Type:            string(tx.Type),
		Date:            tx.Date.Format(time.RFC3339),
		Amount:          tx.Amount,
		Description:     tx.Description,
		SourceId:        srcId,
		SourceName:      srcName,
		DestinationId:   destId,
		DestinationName: destName,
		CategoryId:      catId,
		CategoryName:    tx.Category,
		Tags:            tx.Tags,
		Notes:           tx.Notes,
		ExternalId:      tx.ID,
	}

	// For withdrawals, Destination is the expense account (Counterparty)
	if tx.Type == models.TypeWithdrawal && split.DestinationName == "" {
		split.DestinationName = tx.Counterparty
	}
	// For deposits, Source is the income account (Counterparty)
	if tx.Type == models.TypeDeposit && split.SourceName == "" {
		split.SourceName = tx.Counterparty
	}

	payload := TransactionStore{
		ErrorIfDuplicateHash: false,
		ApplyRules:           true,
		Firehooks:            true,
		Transactions:         []TransactionSplit{split},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/transactions", cfg.FireflyURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.FireflyToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("failed to submit transaction, status: %s, response: %v", resp.Status, errResp)
	}

	return nil
}

func GetCategories(cfg config.Config) ([]Category, error) {
	if cfg.FireflyURL == "" || cfg.FireflyToken == "" {
		return nil, fmt.Errorf("firefly URL or Token is not configured")
	}

	url := fmt.Sprintf("%s/api/v1/categories", cfg.FireflyURL)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.FireflyToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch categories, status: %s", resp.Status)
	}

	var result struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var categories []Category
	for _, item := range result.Data {
		categories = append(categories, Category{ID: item.ID, Name: item.Attributes.Name})
	}
	return categories, nil
}

func GetAccounts(cfg config.Config) ([]Account, error) {
	if cfg.FireflyURL == "" || cfg.FireflyToken == "" {
		return nil, fmt.Errorf("firefly URL or Token is not configured")
	}

	// Fetch asset, expense, revenue accounts
	types := []string{"asset", "expense", "revenue"}
	var accounts []Account

	client := &http.Client{Timeout: 10 * time.Second}

	for _, t := range types {
		url := fmt.Sprintf("%s/api/v1/accounts?type=%s", cfg.FireflyURL, t)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.FireflyToken)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		
		if resp.StatusCode == 200 {
			var result struct {
				Data []struct {
					ID         string `json:"id"`
					Attributes struct {
						Name string `json:"name"`
						Type string `json:"type"`
					} `json:"attributes"`
				} `json:"data"`
			}
			json.NewDecoder(resp.Body).Decode(&result)
			for _, item := range result.Data {
				accounts = append(accounts, Account{
					ID:   item.ID,
					Name: item.Attributes.Name,
					Type: item.Attributes.Type,
				})
			}
		}
		resp.Body.Close()
	}

	return accounts, nil
}

// ExistsTransactionByExternalId checks if a transaction with the given external_id
// already exists in Firefly III using the search API.
func ExistsTransactionByExternalId(externalId string, cfg config.Config) (bool, error) {
	if externalId == "" {
		return false, nil
	}

	client := &http.Client{Timeout: 30 * time.Second}

	url := fmt.Sprintf("%s/api/v1/search/transactions?query=external_id_is:%s", cfg.FireflyURL, externalId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.FireflyToken)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("search API returned status: %s", resp.Status)
	}

	var result struct {
		Data []interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return len(result.Data) > 0, nil
}
