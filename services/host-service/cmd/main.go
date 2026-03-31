package main

import (
	"cmdb-v2/pkg/db"
	"cmdb-v2/pkg/logging"
	"cmdb-v2/pkg/models"
	"cmdb-v2/services/host-service/internal/handlers"
	"cmdb-v2/services/host-service/internal/router"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := logging.Init("host-service"); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:rootpassword@tcp(mysql:3306)/cmdb_resource?charset=utf8mb4&parseTime=True&loc=Local"
	}
	database := db.InitMySQLAndMigrate(dsn, &models.Host{}, &models.AuditLog{})
	r := gin.Default()
	router.Init(r, handlers.New(database), database)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("host-service starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
