package server

import (
	"context"
	"log"
	"time"
	"vaultgo/config"
	"vaultgo/docs"
	"vaultgo/internal/model"
	"vaultgo/internal/router"
	"vaultgo/pkg/middleware"
	"vaultgo/pkg/mongoclient"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func Run() {
	cfg := config.Load()

	// Connect to MongoDB
	client := mongoclient.Connect(cfg.MongoURI, model.NewDecimalRegistry())
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("failed to disconnect mongo: %v", err)
		}
	}()

	// Setup Swagger
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Version = "0.1.0"

	// Setup Gin
	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(middleware.CORSMiddleware())

	db := client.Database(cfg.DBName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Wire up routes
	router.Routes(ctx, r, db, client)

	log.Printf("starting server on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	log.Println("server stopped")
}
