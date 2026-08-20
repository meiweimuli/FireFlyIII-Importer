package parser

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"firefly-importer/models"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ParseAlipayCSV reads Alipay CSV file (GBK encoded) and returns a list of standardized Transactions
func ParseAlipayCSV(data []byte, externalIdField string) ([]models.Transaction, error) {
	// Convert GBK to UTF-8
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	
	// Read all lines to find where the actual header starts
	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode GBK: %w", err)
	}

	lines := strings.Split(string(buf), "\n")
	var csvData string
	headerFound := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "交易时间,") {
			headerFound = true
		}
		if headerFound {
			if line != "" {
				csvData += line + "\n"
			}
		}
	}

	if !headerFound {
		return nil, fmt.Errorf("could not find CSV header row in Alipay file")
	}

	csvReader := csv.NewReader(strings.NewReader(csvData))
	csvReader.TrimLeadingSpace = true
	// Alipay might have trailing commas in headers
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("no data rows found")
	}

	headers := records[0]
	// Clean headers
	for i, h := range headers {
		headers[i] = strings.TrimSpace(h)
	}

	// Helper to find column index
	getIdx := func(name string) int {
		for i, h := range headers {
			if h == name {
				return i
			}
		}
		return -1
	}

	timeIdx := getIdx("交易时间")
	categoryIdx := getIdx("交易分类")
	counterpartyIdx := getIdx("交易对方")
	descIdx := getIdx("商品说明")
	typeIdx := getIdx("收/支")
	amountIdx := getIdx("金额")
	sourceIdx := getIdx("收/付款方式")
	statusIdx := getIdx("交易状态")
	
	idField := "交易订单号"
	if externalIdField != "" {
		idField = externalIdField
	}
	idIdx := getIdx(idField)
	notesIdx := getIdx("备注")

	if timeIdx == -1 || amountIdx == -1 {
		return nil, fmt.Errorf("missing required columns in Alipay CSV (need time and amount)")
	}

	var transactions []models.Transaction

	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) == 0 || row[0] == "" {
			continue
		}

		// Check status
		status := ""
		if statusIdx != -1 && len(row) > statusIdx {
			status = strings.TrimSpace(row[statusIdx])
		}
		// Do not skip any transactions by default, let the user's rule engine handle them

		// Parse time (using China Standard Time / UTC+8)
		timeStr := strings.TrimSpace(row[timeIdx])
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, loc)
		if err != nil {
			// fallback
			t = time.Now()
		}

		// Type
		inOut := ""
		if typeIdx != -1 && len(row) > typeIdx {
			inOut = strings.TrimSpace(row[typeIdx])
		}

		txType := models.TypeWithdrawal
		if inOut == "收入" {
			txType = models.TypeDeposit
		} else if inOut == "不计收支" {
			// Usually internal transfers or balance changes, map based on logic or leave as withdrawal for now
			txType = models.TypeWithdrawal
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
			ID:                 safeGetRow(row, idIdx),
			Date:               t,
			Type:               txType,
			Amount:             strings.TrimSpace(row[amountIdx]),
			Description:        strings.TrimSpace(row[descIdx]),
			Counterparty:       strings.TrimSpace(row[counterpartyIdx]),
			Category:           strings.TrimSpace(row[categoryIdx]),
			PaymentMethod:      strings.TrimSpace(row[sourceIdx]),
			Status:             status,
			TransactionID:      safeGetRow(row, idIdx), // use the configured ID field
			AssetAccount:       "", // Let rule engine set this later
			OpposingAccount:    "",
			IsOutflow:          txType == models.TypeWithdrawal,
			Notes:              strings.TrimSpace(row[notesIdx]),
			RawData:            string(rawBytes),
			Source:             "Alipay",
		}

		// Apply mapping logic later
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func safeGetRow(row []string, idx int) string {
	if idx != -1 && len(row) > idx {
		return strings.TrimSpace(row[idx])
	}
	return ""
}
