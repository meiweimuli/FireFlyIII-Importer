package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type RuleCondition struct {
	// Leaf node fields
	MatchField string `json:"matchField,omitempty"`
	MatchType  string `json:"matchType,omitempty"`
	MatchValue string `json:"matchValue,omitempty"`
	Negate     bool   `json:"negate,omitempty"`

	// Group node fields
	IsGroup        bool            `json:"isGroup"`
	ConditionLogic string          `json:"conditionLogic,omitempty"`
	Conditions     []RuleCondition `json:"conditions,omitempty"`
}

type MappingRule struct {
	Name           string          `json:"name"`
	IsRuleGroup    bool            `json:"isRuleGroup"`
	Rules          []MappingRule   `json:"rules,omitempty"`
	ExcludeAlipay  bool            `json:"excludeAlipay"`
	ExcludeWechat  bool            `json:"excludeWechat"`
	ConditionLogic string          `json:"conditionLogic"` // "AND" or "OR"
	Conditions     []RuleCondition `json:"conditions"`
	Ignore         bool   `json:"ignore"`
	ModifyIgnore   bool   `json:"modifyIgnore"`
	SwapAccounts       bool   `json:"swapAccounts"`
	ModifySwapAccounts bool   `json:"modifySwapAccounts"`
	TargetType     string `json:"targetType"` // "withdrawal", "deposit", "transfer", ""
	TargetCategory   string `json:"targetCategory"`
	TargetCategoryId string `json:"targetCategoryId"`
	TargetAsset      string `json:"targetAsset"`
	TargetAssetId    string `json:"targetAssetId"`
	TargetOpposing   string `json:"targetOpposing"`
	TargetOpposingId string `json:"targetOpposingId"`
	TargetDescription string `json:"targetDescription"`
	TargetNotes      string `json:"targetNotes"`
	TargetTags       string `json:"targetTags"` // comma separated
}

type Config struct {
	FireflyURL            string        `json:"fireflyUrl"`
	FireflyToken          string        `json:"fireflyToken"`
	AlipayExternalIdField string        `json:"alipayExternalIdField"`
	WechatExternalIdField string        `json:"wechatExternalIdField"`
	DefaultAssetAccount    string        `json:"defaultAssetAccount"`
	DefaultAssetAccountId  string        `json:"defaultAssetAccountId"`
	// Backward compat: old config files may have these
	DefaultAlipayAccount   string        `json:"defaultAlipayAccount,omitempty"`
	DefaultAlipayAccountId string        `json:"defaultAlipayAccountId,omitempty"`
	DefaultWechatAccount   string        `json:"defaultWechatAccount,omitempty"`
	DefaultWechatAccountId string        `json:"defaultWechatAccountId,omitempty"`
	DefaultOpposingAccount string        `json:"defaultOpposingAccount"`
	DefaultOpposingId      string        `json:"defaultOpposingId"`
	GlobalTags            string        `json:"globalTags"`
	DeduplicateByExternalId bool        `json:"deduplicateByExternalId"`
	MappingRules          []MappingRule `json:"mappingRules"`
}

var (
	AppConfig Config
	mu        sync.RWMutex
	cfgDir    = "data"
	cfgFile   = "config.json"
)

func cfgPath() string {
	return cfgDir + "/" + cfgFile
}

func SetConfigDir(dir string) {
	cfgDir = dir
}

func LoadConfig() error {
	mu.Lock()
	defer mu.Unlock()

	// Ensure config directory exists
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := os.ReadFile(cfgPath())
	if err != nil {
		if os.IsNotExist(err) {
			AppConfig = Config{}
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &AppConfig)
}

func SaveConfig(cfg Config) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return err
	}
	err = os.WriteFile(cfgPath(), data, 0644)
	if err != nil {
		return err
	}
	AppConfig = cfg
	return nil
}

func BackupConfig() error {
	src := cfgPath()
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to backup
		}
		return err
	}
	backupName := fmt.Sprintf("%s/config-backup-%s.json", cfgDir, time.Now().Format("20060102-150405"))
	return os.WriteFile(backupName, data, 0644)
}

func GetConfig() Config {
	mu.RLock()
	defer mu.RUnlock()
	return AppConfig
}

func LoadPresets() error {
	mu.Lock()
	defer mu.Unlock()

	presets := []MappingRule{
		{
			Name: "美团/饿了么 -> 餐饮",
			ConditionLogic: "OR",
			Conditions: []RuleCondition{
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "美团"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "饿了么"},
			},
			TargetCategory: "餐饮美食",
		},
		{
			Name: "滴滴/单车 -> 交通",
			ConditionLogic: "OR",
			Conditions: []RuleCondition{
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "滴滴"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "哈啰"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "单车"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "公交"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "地铁"},
			},
			TargetCategory: "交通出行",
		},
		{
			Name: "淘宝/京东/拼多多 -> 购物",
			ConditionLogic: "OR",
			Conditions: []RuleCondition{
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "淘宝"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "京东"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "拼多多"},
			},
			TargetCategory: "购物",
		},
		{
			Name: "余额宝收益/理财 -> 投资",
			ConditionLogic: "OR",
			Conditions: []RuleCondition{
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "余额宝"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "理财"},
				{MatchField: "Description", MatchType: "Contains", MatchValue: "收益发放"},
			},
			TargetType: "transfer",
			TargetCategory: "投资理财",
		},
		{
			Name: "国家电网/水电气 -> 居家",
			ConditionLogic: "OR",
			Conditions: []RuleCondition{
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "国家电网"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "燃气"},
				{MatchField: "Counterparty", MatchType: "Contains", MatchValue: "水务"},
			},
			TargetCategory: "居家",
		},
	}

	AppConfig.MappingRules = append(AppConfig.MappingRules, presets...)
	
	// Save to file immediately
	data, err := json.MarshalIndent(AppConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath(), data, 0644)
}
