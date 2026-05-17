package api

import (
	"encoding/json"
	"firefly-importer/config"
	"firefly-importer/models"
	"firefly-importer/parser"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterUploadRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.POST("/upload", handleUpload)
		api.POST("/evaluate", handleEvaluate)
	}
}

func handleUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	var transactions []models.Transaction
	var parseErr error
	cfg := config.GetConfig()

	if strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		// Assuming Alipay CSV
		transactions, parseErr = parser.ParseAlipayCSV(data, cfg.AlipayExternalIdField)
	} else if strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		// Assuming WeChat XLSX
		transactions, parseErr = parser.ParseWeChatXLSX(data, cfg.WechatExternalIdField)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file format. Please upload .csv or .xlsx"})
		return
	}

	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErr.Error()})
		return
	}

	// Apply mapping rules
	isAlipay := strings.HasSuffix(strings.ToLower(file.Filename), ".csv")
	transactions = applyMappingRules(transactions, cfg, isAlipay)

	c.JSON(http.StatusOK, gin.H{
		"message":      "File parsed successfully",
		"transactions": transactions,
	})
}

type EvaluateRequest struct {
	Transactions []models.Transaction `json:"transactions"`
	IsAlipay     bool                 `json:"isAlipay"`
}

func handleEvaluate(c *gin.Context) {
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	cfg := config.GetConfig()
	transactions := applyMappingRules(req.Transactions, cfg, req.IsAlipay)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Rules evaluated successfully",
		"transactions": transactions,
	})
}

func applyMappingRules(transactions []models.Transaction, cfg config.Config, isAlipay bool) []models.Transaction {
	rules := cfg.MappingRules
	
	defaultAccount := cfg.DefaultAssetAccount
	defaultAccountId := cfg.DefaultAssetAccountId
	// Backward compat: fall back to old per-platform fields
	if defaultAccount == "" && defaultAccountId == "" {
		if isAlipay {
			defaultAccount = cfg.DefaultAlipayAccount
			defaultAccountId = cfg.DefaultAlipayAccountId
		} else {
			defaultAccount = cfg.DefaultWechatAccount
			defaultAccountId = cfg.DefaultWechatAccountId
		}
	}

	for i, tx := range transactions {
		// Set default account if configured and not already set
		if defaultAccount != "" || defaultAccountId != "" {
			if tx.AssetAccount == "" && tx.AssetId == "" {
				transactions[i].AssetAccount = defaultAccount
				transactions[i].AssetId = defaultAccountId
			}
		}

		// Apply global tags
		if cfg.GlobalTags != "" {
			tags := strings.Split(cfg.GlobalTags, ",")
			for _, t := range tags {
				if t = strings.TrimSpace(t); t != "" {
					transactions[i].Tags = append(transactions[i].Tags, t)
				}
			}
		}

		var rawMap map[string]string
		if tx.RawData != "" {
			json.Unmarshal([]byte(tx.RawData), &rawMap)
		}

		shouldSwap := false

		applyRuleOrGroup(&transactions[i], tx.Source, rules, rawMap, &shouldSwap)

		// Apply final swap state after all rules have evaluated
		if shouldSwap {
			tempAcc := transactions[i].AssetAccount
			tempId := transactions[i].AssetId
			transactions[i].AssetAccount = transactions[i].OpposingAccount
			transactions[i].AssetId = transactions[i].OpposingId
			transactions[i].OpposingAccount = tempAcc
			transactions[i].OpposingId = tempId
		}
	}
	for i := range transactions {
		if transactions[i].OpposingAccount == "" && transactions[i].OpposingId == "" && cfg.DefaultOpposingAccount != "" {
			transactions[i].OpposingAccount = cfg.DefaultOpposingAccount
			transactions[i].OpposingId = cfg.DefaultOpposingId
		}
	}

	return transactions
}

func applyRuleOrGroup(tx *models.Transaction, source string, rules []config.MappingRule, rawMap map[string]string, shouldSwap *bool) {
	for _, rule := range rules {
		if source == "Alipay" && rule.ExcludeAlipay {
			continue
		}
		if source == "Wechat" && rule.ExcludeWechat {
			continue
		}

		if rule.IsRuleGroup {
			// If group has conditions, evaluate them first; no conditions = pass
			if len(rule.Conditions) > 0 {
				groupMatched := evaluateGroup(rule.ConditionLogic, rule.Conditions, rawMap)
				if !groupMatched {
					continue
				}
			}
			applyRuleOrGroup(tx, source, rule.Rules, rawMap, shouldSwap)
			continue
		}

		if len(rule.Conditions) == 0 {
			continue
		}

		ruleMatched := evaluateGroup(rule.ConditionLogic, rule.Conditions, rawMap)

		if ruleMatched {
			if rule.ModifyIgnore {
				tx.Ignore = rule.Ignore
			}
			if rule.TargetType != "" {
				if rule.TargetType == "withdrawal" {
					tx.Type = models.TypeWithdrawal
				} else if rule.TargetType == "deposit" {
					tx.Type = models.TypeDeposit
				} else if rule.TargetType == "transfer" {
					tx.Type = models.TypeTransfer
				}
			}
			if rule.TargetCategory != "" || rule.TargetCategoryId != "" {
				tx.Category = rule.TargetCategory
				tx.CategoryId = rule.TargetCategoryId
			}
			if rule.TargetAsset != "" || rule.TargetAssetId != "" {
				tx.AssetAccount = rule.TargetAsset
				tx.AssetId = rule.TargetAssetId
			}
			if rule.TargetOpposing != "" || rule.TargetOpposingId != "" {
				tx.OpposingAccount = rule.TargetOpposing
				tx.OpposingId = rule.TargetOpposingId
			}
			if rule.ModifySwapAccounts {
				*shouldSwap = rule.SwapAccounts
			}

			applyTemplate := func(template string) string {
				if template == "" {
					return ""
				}
				result := template
				var rMap map[string]string
				if tx.RawData != "" {
					if err := json.Unmarshal([]byte(tx.RawData), &rMap); err == nil {
						for k, v := range rMap {
							result = strings.ReplaceAll(result, "${"+k+"}", v)
						}
					}
				}
				return result
			}

			if rule.TargetDescription != "" {
				tx.Description = applyTemplate(rule.TargetDescription)
			}
			if rule.TargetNotes != "" {
				tx.Notes = applyTemplate(rule.TargetNotes)
			}

			if rule.TargetTags != "" {
				tags := strings.Split(rule.TargetTags, ",")
				for _, t := range tags {
					if t = strings.TrimSpace(t); t != "" {
						tx.Tags = append(tx.Tags, t)
					}
				}
			}
		}
	}
}

func evaluateGroup(logic string, conditions []config.RuleCondition, rawMap map[string]string) bool {
	if len(conditions) == 0 {
		return false
	}
	if logic == "OR" {
		for _, cond := range conditions {
			if evaluateCondition(cond, rawMap) {
				return true
			}
		}
		return false
	} else {
		// Default to AND
		for _, cond := range conditions {
			if !evaluateCondition(cond, rawMap) {
				return false
			}
		}
		return true
	}
}

func evaluateCondition(cond config.RuleCondition, rawMap map[string]string) bool {
	if cond.IsGroup {
		result := evaluateGroup(cond.ConditionLogic, cond.Conditions, rawMap)
		if cond.Negate {
			return !result
		}
		return result
	}

	fieldVal := rawMap[cond.MatchField]
	fieldValLower := strings.ToLower(fieldVal)
	pattern := strings.ToLower(cond.MatchValue)

	var result bool
	switch cond.MatchType {
	case "Equals":
		result = fieldValLower == pattern
	case "Regex":
		re, err := regexp.Compile(cond.MatchValue)
		if err == nil {
			result = re.MatchString(fieldVal)
		}
	case ">":
		amt, _ := strconv.ParseFloat(fieldVal, 64)
		patAmt, _ := strconv.ParseFloat(cond.MatchValue, 64)
		result = amt > patAmt
	case "<":
		amt, _ := strconv.ParseFloat(fieldVal, 64)
		patAmt, _ := strconv.ParseFloat(cond.MatchValue, 64)
		result = amt < patAmt
	default: // "Contains"
		result = strings.Contains(fieldValLower, pattern)
	}

	if cond.Negate {
		return !result
	}
	return result
}
