package api

import (
	"encoding/json"
	"firefly-importer/config"
	"firefly-importer/firefly"
	"firefly-importer/models"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterImportRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.POST("/import", handleImport)
	}
}

type ImportRequest struct {
	Transactions []models.Transaction `json:"transactions"`
}

type ImportProgress struct {
	Index       int    `json:"index"`
	Total       int    `json:"total"`
	TxId        string `json:"txId"`
	Description string `json:"description"`
	Status      string `json:"status"` // "success", "error", "skipped", "duplicate"
	Message     string `json:"message"`
}

func handleImport(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	cfg := config.GetConfig()
	if cfg.FireflyURL == "" || cfg.FireflyToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Firefly URL or Token is not configured"})
		return
	}

	// Filter out ignored transactions
	var toImport []models.Transaction
	for _, tx := range req.Transactions {
		if !tx.Ignore {
			toImport = append(toImport, tx)
		}
	}

	// Sort by date ascending (earliest first)
	sort.Slice(toImport, func(i, j int) bool {
		return toImport[i].Date.Before(toImport[j].Date)
	})

	total := len(toImport)

	// Generate import batch tag: import-20260517-211500
	importTag := "import-" + time.Now().Format("20060102-150405")
	for i := range toImport {
		toImport[i].Tags = append([]string{importTag}, toImport[i].Tags...)
	}

	// Set up SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	for i, tx := range toImport {
		// Stop if client disconnected
		select {
		case <-c.Request.Context().Done():
			fmt.Printf("[import] client disconnected, stopping at %d/%d\n", i, total)
			return
		default:
		}

		progress := ImportProgress{
			Index:       i,
			Total:       total,
			TxId:        tx.ID,
			Description: tx.Description,
		}

		// Server-side dedup check: query Firefly III for existing external_id
		if cfg.DeduplicateByExternalId && tx.ID != "" {
			exists, err := firefly.ExistsTransactionByExternalId(tx.ID, cfg)
			if err != nil {
				// If search fails, log but continue with import
				fmt.Printf("[dedup] search failed for %s: %v, proceeding with import\n", tx.ID, err)
			} else if exists {
				progress.Status = "duplicate"
				progress.Message = "Skipped: already exists in Firefly III"
				sendSSE(c, progress)
				continue
			}
		}

		err := firefly.SubmitTransaction(tx, cfg)
		if err != nil {
			progress.Status = "error"
			progress.Message = fmt.Sprintf("Error: %v", err)
		} else {
			progress.Status = "success"
			progress.Message = "Imported successfully"
		}

		sendSSE(c, progress)
	}

	// Send final completion event
	data, _ := json.Marshal(gin.H{"done": true, "total": total})
	fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", string(data))
	c.Writer.Flush()
}

func sendSSE(c *gin.Context, progress ImportProgress) {
	data, _ := json.Marshal(progress)
	fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
	c.Writer.Flush()
}
