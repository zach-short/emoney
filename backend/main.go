package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zachmshort/emoney-backend/config"
	"github.com/zachmshort/emoney-backend/controllers"
	"github.com/zachmshort/emoney-backend/routes"
)

func main() {

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"https://emoney.club",
			"https://www.emoney.club",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Upgrade", "Connection"},
		AllowWebSockets:  true,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	config.ConnectDB()
	// Logged, not fatal: the offer routes work without their indexes, only
	// slower, and a startup that dies on an index error takes every room
	// with it on the one process this app runs (HANDOFF.md invariant 1).
	if err := controllers.EnsureOfferIndexes(); err != nil {
		log.Printf("Failed to create the Offer indexes: %v", err)
	}
	routes.Routes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
