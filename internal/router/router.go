package router

import (
	"context"
	"fmt"
	"net/http"
	"vaultgo/internal/dto"
	"vaultgo/internal/handler"
	"vaultgo/internal/repository"
	"vaultgo/internal/service"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Routes(ctx context.Context, r *gin.Engine, db *mongo.Database, client *mongo.Client) {
	r.GET("/", Home)
	r.GET("/health", HealthCheck)
	r.GET("/documentation/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)

	walletRepo.CreateIndexes(ctx)
	ledgerRepo.CreateIndexes(ctx)

	walletSvc := service.NewWalletService(walletRepo, ledgerRepo, client)
	walletHandler := handler.NewWalletHandler(walletSvc)

	walletHandler.RegisterRoutes(r)
}

func Home(c *gin.Context) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>VaultGo Documentation</title>
  </head>
  <body>
    <h1>Welcome to VaultGo</h1>
    <p><a href="/health">health check</a></p>
    <p><a href="/documentation/index.html">docs</a></p>
  </body>
</html>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, dto.Response{
		Status:  "success",
		Message: "VaultGo server is healthy",
	})
}
