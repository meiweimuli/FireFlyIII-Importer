package main

import (
	"firefly-importer/api"
	"firefly-importer/config"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Support custom config directory via env (for Docker volume mounting)
	if dir := os.Getenv("CONFIG_DIR"); dir != "" {
		config.SetConfigDir(dir)
	}

	if err := config.LoadConfig(); err != nil {
		log.Printf("Warning: failed to load config: %v", err)
	}

	r := gin.Default()

	// Enable CORS for frontend development
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(corsConfig))

	api.RegisterRoutes(r)
	api.RegisterUploadRoutes(r)
	api.RegisterImportRoutes(r)

	// Serve frontend static files
	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/favicon.ico", "./frontend/dist/favicon.ico")
	r.StaticFile("/favicon.svg", "./frontend/dist/favicon.svg")
	r.StaticFile("/vite.svg", "./frontend/dist/vite.svg")
	r.NoRoute(func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
