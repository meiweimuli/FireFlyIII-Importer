package api

import (
	"firefly-importer/config"
	"firefly-importer/firefly"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/config", getConfig)
		api.POST("/config", saveConfig)
		api.POST("/config/import", importConfig)
		api.POST("/config/presets", loadPresets)
		api.GET("/firefly/categories", getCategories)
		api.GET("/firefly/accounts", getAccounts)
	}
}

func getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, config.GetConfig())
}

func saveConfig(c *gin.Context) {
	var cfg config.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := config.SaveConfig(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Configuration saved successfully"})
}

func loadPresets(c *gin.Context) {
	err := config.LoadPresets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Presets loaded successfully"})
}

func importConfig(c *gin.Context) {
	var cfg config.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config JSON: " + err.Error()})
		return
	}
	// Backup current config
	if err := config.BackupConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to backup config: " + err.Error()})
		return
	}
	// Save imported config
	if err := config.SaveConfig(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save imported config: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Config imported successfully. Previous config backed up."})
}

func getCategories(c *gin.Context) {
	cfg := config.GetConfig()
	categories, err := firefly.GetCategories(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func getAccounts(c *gin.Context) {
	cfg := config.GetConfig()
	accounts, err := firefly.GetAccounts(cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accounts)
}
