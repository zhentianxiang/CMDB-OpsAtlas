package main

import (
	"cmdb-v2/pkg/db"
	"cmdb-v2/pkg/logging"
	"cmdb-v2/pkg/models"
	"cmdb-v2/services/topology-service/internal/handlers"
	"cmdb-v2/services/topology-service/internal/router"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := logging.Init("topology-service"); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:rootpassword@tcp(mysql:3306)/cmdb_resource?charset=utf8mb4&parseTime=True&loc=Local"
	}
	database := db.InitMySQLAndMigrate(
		dsn,
		&models.Host{},
		&models.App{},
		&models.Dependency{},
	)
	r := gin.Default()
	router.Init(r, handlers.New(database))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("topology-service starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
