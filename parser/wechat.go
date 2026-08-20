package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"firefly-importer/models"

	"github.com/xuri/excelize/v2"
)

// ParseWeChatXLSX reads WeChat XLSX file and returns a list of standardized Transactions
func ParseWeChatXLSX(data []byte, externalIdField string) ([]models.Transaction, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	var headers []string
	headerIdx := -1

	// Find header row
	for i, row := range rows {
		if len(row) > 0 && strings.Contains(row[0], "交易时间") {
			headers = row
			headerIdx = i
			break
		}
	}

	if headerIdx == -1 {
		return nil, fmt.Errorf("could not find header row in WeChat file")
	}

	// Clean headers
	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}

	getIdx := func(name string) int {
		for i, h := range headers {
			if strings.Contains(h, name) {
				return i
			}
		}
		return -1
	}

	timeIdx := getIdx("交易时间")
	typeIdx := getIdx("收/支")
	amountIdx := getIdx("金额")
	counterpartyIdx := getIdx("交易对方")
	descIdx := getIdx("商品")
	categoryIdx := getIdx("交易类型")
	sourceIdx := getIdx("支付方式")
	statusIdx := getIdx("当前状态")
	
	idField := "交易单号"
	if externalIdField != "" {
		idField = externalIdField
	}
	idIdx := getIdx(idField)
	notesIdx := getIdx("备注")

	if timeIdx == -1 || amountIdx == -1 {
		return nil, fmt.Errorf("missing required columns in WeChat XLSX (need time and amount)")
	}

	var transactions []models.Transaction

	for i := headerIdx + 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 || row[0] == "" {
			continue
		}

		status := ""
		if statusIdx != -1 && len(row) > statusIdx {
			status = strings.TrimSpace(row[statusIdx])
		}
		if strings.Contains(status, "已退款") || strings.Contains(status, "支付失败") {
			continue
		}

		timeStr := ""
		if len(row) > timeIdx {
			timeStr = strings.TrimSpace(row[timeIdx])
		}
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, loc)
		if err != nil {
			t = time.Now()
		}

		inOut := ""
		if typeIdx != -1 && len(row) > typeIdx {
			inOut = strings.TrimSpace(row[typeIdx])
		}

		txType := models.TypeWithdrawal
		if inOut == "收入" {
			txType = models.TypeDeposit
		}

		amount := ""
		if len(row) > amountIdx {
			amount = strings.TrimSpace(row[amountIdx])
			// WeChat amounts have ¥ prefix
			amount = strings.ReplaceAll(amount, "¥", "")
		}

		// Build raw JSON
		rawMap := make(map[string]string)
		for j, h := range headers {
			if j < len(row) {
				rawMap[h] = row[j]
			}
		}
		rawBytes, _ := json.Marshal(rawMap)

		tx := models.Transaction{
			ID:                 safeGet(row, idIdx),
			Date:               t,
			Type:               txType,
			Amount:             amount,
			Description:        safeGet(row, descIdx),
			Counterparty:       safeGet(row, counterpartyIdx),
			Category:           safeGet(row, categoryIdx),
			PaymentMethod:      safeGet(row, sourceIdx),
			Status:             status,
			TransactionID:      safeGet(row, idIdx),
			AssetAccount:       "", // Let rule engine set this later
			OpposingAccount:    "",
			IsOutflow:          txType == models.TypeWithdrawal,
			Notes:              safeGet(row, notesIdx),
			RawData:            string(rawBytes),
			Source:             "Wechat",
		}

		// Clean up fields
		if tx.Notes == "/" {
			tx.Notes = ""
		}
		if tx.Counterparty == "/" {
			tx.Counterparty = ""
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func safeGet(row []string, idx int) string {
	if idx != -1 && len(row) > idx {
		return strings.TrimSpace(row[idx])
	}
	return ""
}
